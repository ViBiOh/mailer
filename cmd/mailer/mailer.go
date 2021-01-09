package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/httputils/v3/pkg/alcotest"
	"github.com/ViBiOh/httputils/v3/pkg/cors"
	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/httputils"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/model"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
	"github.com/ViBiOh/mailer/pkg/amqphandler"
	"github.com/ViBiOh/mailer/pkg/httphandler"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

const (
	fixturesPath = "/fixtures"
	renderPath   = "/render"
)

func main() {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com"))
	corsConfig := cors.Flags(fs, "cors")

	amqpConfig := amqphandler.Flags(fs, "amqp")
	smtpConfig := smtp.Flags(fs, "smtp")
	mjmlConfig := mjml.Flags(fs, "mjml")
	mailerConfig := mailer.Flags(fs, "mailer")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	mjmlApp := mjml.New(mjmlConfig)
	senderApp := smtp.New(smtpConfig)
	mailerApp := mailer.New(mailerConfig, mjmlApp, senderApp)
	amqpApp := amqphandler.New(amqpConfig, mailerApp)

	httputilsApp := httputils.New(serverConfig)
	go amqpApp.Start(httputilsApp.GetDone())

	httputilsApp.ListenAndServe(httphandler.New(mailerApp).Handler(), []model.Pinger{amqpApp.Ping}, prometheus.New(prometheusConfig).Middleware, owasp.New(owaspConfig).Middleware, cors.New(corsConfig).Middleware)
}
