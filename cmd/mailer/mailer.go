package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/alcotest"
	"github.com/ViBiOh/httputils/v3/pkg/cors"
	"github.com/ViBiOh/httputils/v3/pkg/httputils"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
	"github.com/ViBiOh/httputils/v3/pkg/swagger"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/render"
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
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")
	corsConfig := cors.Flags(fs, "cors")
	swaggerConfig := swagger.Flags(fs, "swagger")

	smtpConfig := smtp.Flags(fs, "smtp")
	mailjetConfig := mailjet.Flags(fs, "mailjetApi")
	mjmlConfig := mjml.Flags(fs, "mjml")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	server := httputils.New(serverConfig)
	mjmlApp := mjml.New(mjmlConfig)

	senderApp := mailjet.New(mailjetConfig)
	if senderApp == nil {
		senderApp = smtp.New(smtpConfig)
	}

	renderApp := render.New(mjmlApp, senderApp)
	prometheusApp := prometheus.New(prometheusConfig)

	swaggerApp, err := swagger.New(swaggerConfig, server.Swagger, prometheusApp.Swagger, renderApp.Swagger, fixtures.Swagger)
	logger.Fatal(err)

	renderHandler := http.StripPrefix(renderPath, renderApp.Handler())
	fixtureHandler := http.StripPrefix(fixturesPath, fixtures.Handler())
	swaggerHandler := swaggerApp.Handler()

	mailerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, renderPath) {
			renderHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, fixturesPath) {
			fixtureHandler.ServeHTTP(w, r)
			return
		}

		swaggerHandler.ServeHTTP(w, r)
	})

	server.Middleware(prometheusApp.Middleware)
	server.Middleware(owasp.New(owaspConfig).Middleware)
	server.Middleware(cors.New(corsConfig).Middleware)
	server.ListenServeWait(mailerHandler)
}
