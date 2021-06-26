package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/cors"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/mailer/pkg/amqphandler"
	"github.com/ViBiOh/mailer/pkg/httphandler"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

func main() {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "")
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", 9090), flags.NewOverride("IdleTimeout", "10s"), flags.NewOverride("ShutdownTimeout", "5s"))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com https://ketchup.vibioh.fr/images/"))
	corsConfig := cors.Flags(fs, "cors")

	amqpConfig := amqphandler.Flags(fs, "amqp")
	smtpConfig := smtp.Flags(fs, "smtp")
	mjmlConfig := mjml.Flags(fs, "mjml")
	mailerConfig := mailer.Flags(fs, "")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)

	mjmlApp := mjml.New(mjmlConfig)
	senderApp := smtp.New(smtpConfig)
	mailerApp := mailer.New(mailerConfig, mjmlApp, senderApp)

	amqpApp, err := amqphandler.New(amqpConfig, mailerApp)
	if err != nil {
		logger.Error("unable to create amqp: %s", err)
	}

	healthApp := health.New(healthConfig, amqpApp.Ping)

	go amqpApp.Start(healthApp.Done())

	appHandler := httphandler.New(mailerApp).Handler()

	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(appHandler, healthApp, recoverer.Middleware, prometheusApp.Middleware, owasp.New(owaspConfig).Middleware, cors.New(corsConfig).Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done())
}
