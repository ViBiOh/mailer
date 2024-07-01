package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/cors"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/pprof"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

type configuration struct {
	logger    *logger.Config
	alcotest  *alcotest.Config
	telemetry *telemetry.Config
	pprof     *pprof.Config
	health    *health.Config

	server *server.Config
	owasp  *owasp.Config
	cors   *cors.Config

	amqp        *amqp.Config
	amqphandler *amqphandler.Config

	smtp   *smtp.Config
	mjml   *mjml.Config
	mailer *mailer.Config
}

func newConfig() configuration {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	config := configuration{
		logger:    logger.Flags(fs, "logger"),
		alcotest:  alcotest.Flags(fs, ""),
		telemetry: telemetry.Flags(fs, "telemetry"),
		pprof:     pprof.Flags(fs, "pprof"),
		health:    health.Flags(fs, ""),

		server: server.Flags(fs, ""),
		owasp:  owasp.Flags(fs, "", flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com https://ketchup.vibioh.fr/images/")),
		cors:   cors.Flags(fs, "cors"),

		amqp:        amqp.Flags(fs, "amqp"),
		amqphandler: amqphandler.Flags(fs, "amqp", flags.NewOverride("Exchange", "mailer"), flags.NewOverride("Queue", "mailer")),

		smtp:   smtp.Flags(fs, "smtp"),
		mjml:   mjml.Flags(fs, "mjml"),
		mailer: mailer.Flags(fs, ""),
	}

	_ = fs.Parse(os.Args[1:])

	return config
}
