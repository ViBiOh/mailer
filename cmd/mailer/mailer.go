package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/cors"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/pprof"
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
	pprofConfig := pprof.Flags(fs, "pprof")
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com https://ketchup.vibioh.fr/images/"))
	corsConfig := cors.Flags(fs, "cors")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqHandlerConfig := amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "mailer"), flags.NewOverride("Queue", "mailer"))
	smtpConfig := smtp.Flags(fs, "smtp")
	mjmlConfig := mjml.Flags(fs, "mjml")
	mailerConfig := mailer.Flags(fs, "")

	_ = fs.Parse(os.Args[1:])

	alcotest.DoAndExit(alcotestConfig)

	logger.Init(loggerConfig)

	ctx := context.Background()

	healthService := health.New(ctx, healthConfig)

	telemetryApp, err := telemetry.New(ctx, tracerConfig)
	logger.FatalfOnErr(ctx, err, "telemetry")

	defer telemetryApp.Close(ctx)

	logger.AddOpenTelemetryToDefaultLogger(telemetryApp)
	request.AddOpenTelemetryToDefaultClient(telemetryApp.MeterProvider(), telemetryApp.TracerProvider())

	service, version, env := telemetryApp.GetServiceVersionAndEnv()
	pprofService := pprof.New(pprofConfig, service, version, env)

	go pprofService.Start(healthService.DoneCtx())

	appServer := server.New(appServerConfig)

	mjmlService := mjml.New(mjmlConfig, telemetryApp.MeterProvider(), telemetryApp.TracerProvider())
	senderService := smtp.New(smtpConfig, telemetryApp.MeterProvider(), telemetryApp.TracerProvider())
	mailerService := mailer.New(mailerConfig, mjmlService, senderService, telemetryApp.MeterProvider(), telemetryApp.TracerProvider())

	amqpClient, err := amqp.New(amqpConfig, telemetryApp.MeterProvider(), telemetryApp.TracerProvider())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		slog.LogAttrs(ctx, slog.LevelError, "create amqp", slog.Any("error", err))
		os.Exit(1)
	}

	amqpService, err := amqphandler.New(amqHandlerConfig, amqpClient, telemetryApp.MeterProvider(), telemetryApp.TracerProvider(), mailerService.AmqpHandler)
	logger.FatalfOnErr(ctx, err, "create amqp handler")

	go amqpService.Start(healthService.DoneCtx())

	appHandler := httphandler.New(mailerService, telemetryApp.TracerProvider()).Handler()

	go appServer.Start(healthService.EndCtx(), httputils.Handler(appHandler, healthService, recoverer.Middleware, telemetryApp.Middleware("http"), owasp.New(owaspConfig).Middleware, cors.New(corsConfig).Middleware))

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
