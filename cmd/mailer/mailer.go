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
	"github.com/ViBiOh/httputils/v3/pkg/model"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
	"github.com/ViBiOh/mailer/pkg/fixtures"
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
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")
	corsConfig := cors.Flags(fs, "cors")

	smtpConfig := smtp.Flags(fs, "smtp")
	mjmlConfig := mjml.Flags(fs, "mjml")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	mjmlApp := mjml.New(mjmlConfig)
	senderApp := smtp.New(smtpConfig)

	renderApp := render.New(mjmlApp, senderApp)
	prometheusApp := prometheus.New(prometheusConfig)

	renderHandler := http.StripPrefix(renderPath, renderApp.Handler())
	fixtureHandler := http.StripPrefix(fixturesPath, fixtures.Handler())

	mailerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, renderPath) {
			renderHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, fixturesPath) {
			fixtureHandler.ServeHTTP(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	httputils.New(serverConfig).ListenAndServe(mailerHandler, []model.Middleware{
		prometheusApp.Middleware,
		owasp.New(owaspConfig).Middleware,
		cors.New(corsConfig).Middleware,
	})
}
