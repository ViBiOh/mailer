package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

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
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/mailer/pkg/httphandler"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

func main() {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "")
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", uint(9090)), flags.NewOverride("IdleTimeout", 10*time.Second), flags.NewOverride("ShutdownTimeout", 5*time.Second))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	tracerConfig := tracer.Flags(fs, "tracer")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com https://ketchup.vibioh.fr/images/"))
	corsConfig := cors.Flags(fs, "cors")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqHandlerConfig := amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "mailer"), flags.NewOverride("Queue", "mailer"))
	smtpConfig := smtp.Flags(fs, "smtp")
	mjmlConfig := mjml.Flags(fs, "mjml")
	mailerConfig := mailer.Flags(fs, "")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	tracerApp, err := tracer.New(tracerConfig)
	logger.Fatal(err)
	defer tracerApp.Close()
	request.AddTracerToDefaultClient(tracerApp.GetProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)

	mjmlApp := mjml.New(mjmlConfig, prometheusApp.Registerer())
	senderApp := smtp.New(smtpConfig, prometheusApp.Registerer())
	mailerApp := mailer.New(mailerConfig, mjmlApp, senderApp, prometheusApp.Registerer(), tracerApp)

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		logger.Fatal(err)
	}

	amqpApp, err := amqphandler.New(amqHandlerConfig, amqpClient, mailerApp.AmqpHandler)
	if err != nil {
		logger.Error("create amqp handler: %s", err)
	}

	healthApp := health.New(healthConfig)

	go amqpApp.Start(healthApp.Done())

	appHandler := httphandler.New(mailerApp, tracerApp).Handler()

	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(appHandler, healthApp, recoverer.Middleware, prometheusApp.Middleware, tracerApp.Middleware, owasp.New(owaspConfig).Middleware, cors.New(corsConfig).Middleware))

	healthApp.WaitForTermination(getDoneChan(appServer.Done(), amqpClient, amqpApp))
	server.GracefulWait(appServer.Done(), promServer.Done(), amqpApp.Done())
}

func getDoneChan(httpDone <-chan struct{}, amqpClient *amqp.Client, amqpApp amqphandler.App) <-chan struct{} {
	if amqpClient == nil {
		return httpDone
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		select {
		case <-httpDone:
		case <-amqpApp.Done():
		}
	}()

	return done
}
