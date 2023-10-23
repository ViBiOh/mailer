package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/cors"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/mailer/pkg/httphandler"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

func main() {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	appServerConfig := server.Flags(fs, "")
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	tracerConfig := telemetry.Flags(fs, "telemetry")
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com https://ketchup.vibioh.fr/images/"))
	corsConfig := cors.Flags(fs, "cors")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqHandlerConfig := amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "mailer"), flags.NewOverride("Queue", "mailer"))
	smtpConfig := smtp.Flags(fs, "smtp")
	mjmlConfig := mjml.Flags(fs, "mjml")
	mailerConfig := mailer.Flags(fs, "")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	alcotest.DoAndExit(alcotestConfig)

	logger.Init(loggerConfig)

	ctx := context.Background()

	telemetryService, err := telemetry.New(ctx, tracerConfig)
	if err != nil {
		slog.Error("telemetry", "err", err)
		os.Exit(1)
	}

	defer telemetryService.Close(ctx)
	request.AddOpenTelemetryToDefaultClient(telemetryService.MeterProvider(), telemetryService.TracerProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)

	mjmlService := mjml.New(mjmlConfig, telemetryService.MeterProvider(), telemetryService.TracerProvider())
	senderService := smtp.New(smtpConfig, telemetryService.MeterProvider(), telemetryService.TracerProvider())
	mailerService := mailer.New(mailerConfig, mjmlService, senderService, telemetryService.MeterProvider(), telemetryService.TracerProvider())

	amqpClient, err := amqp.New(amqpConfig, telemetryService.MeterProvider(), telemetryService.TracerProvider())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		slog.Error("create amqp", "err", err)
		os.Exit(1)
	}

	amqpService, err := amqphandler.New(amqHandlerConfig, amqpClient, telemetryService.MeterProvider(), telemetryService.TracerProvider(), mailerService.AmqpHandler)
	if err != nil {
		slog.Error("create amqp handler", "err", err)
		os.Exit(1)
	}

	healthService := health.New(ctx, healthConfig)

	go amqpService.Start(healthService.DoneCtx())

	appHandler := httphandler.New(mailerService, telemetryService.TracerProvider()).Handler()

	go appServer.Start(healthService.EndCtx(), httputils.Handler(appHandler, healthService, recoverer.Middleware, telemetryService.Middleware("http"), owasp.New(owaspConfig).Middleware, cors.New(corsConfig).Middleware))

	healthService.WaitForTermination(getDoneChan(appServer.Done(), amqpClient, amqpService))
	server.GracefulWait(appServer.Done(), amqpService.Done())
}

func getDoneChan(httpDone <-chan struct{}, amqpClient *amqp.Client, amqpService *amqphandler.Service) <-chan struct{} {
	if amqpClient == nil {
		return httpDone
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		select {
		case <-httpDone:
		case <-amqpService.Done():
		}
	}()

	return done
}
