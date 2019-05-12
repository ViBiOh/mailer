package main

import (
	"flag"
	"net/http"
	"os"
	"strings"

	httputils "github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/alcotest"
	"github.com/ViBiOh/httputils/pkg/cors"
	"github.com/ViBiOh/httputils/pkg/gzip"
	"github.com/ViBiOh/httputils/pkg/healthcheck"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/opentracing"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/httputils/pkg/prometheus"
	"github.com/ViBiOh/httputils/pkg/server"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	mailerHealthcheck "github.com/ViBiOh/mailer/pkg/healthcheck"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/render"
)

const (
	fixturesPath = "/fixtures"
	renderPath   = "/render"
	sendPath     = "/send"
)

func main() {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	opentracingConfig := opentracing.Flags(fs, "tracing")
	owaspConfig := owasp.Flags(fs, "")
	corsConfig := cors.Flags(fs, "cors")

	mailjetConfig := mailjet.Flags(fs, "mailjet")
	mjmlConfig := mjml.Flags(fs, "mjml")

	if err := fs.Parse(os.Args[1:]); err != nil {
		logger.Fatal("%+v", err)
	}

	alcotest.DoAndExit(alcotestConfig)

	serverApp, err := httputils.New(serverConfig)
	if err != nil {
		logger.Fatal("%+v", err)
	}

	healthcheckApp := healthcheck.New()
	prometheusApp := prometheus.New(prometheusConfig)
	opentracingApp := opentracing.New(opentracingConfig)
	gzipApp := gzip.New()
	owaspApp := owasp.New(owaspConfig)
	corsApp := cors.New(corsConfig)

	mjmlApp := mjml.New(mjmlConfig)
	mailjetApp := mailjet.New(mailjetConfig)
	renderApp := render.New(mjmlApp, mailjetApp)

	healthcheckApp.NextHealthcheck(mailerHealthcheck.New(mailjetApp).Handler())

	mailjetHandler := mailjetApp.Handler()
	renderHandler := http.StripPrefix(renderPath, renderApp.Handler())
	fixtureHandler := http.StripPrefix(fixturesPath, fixtures.Handler())

	mailerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, renderPath) {
			renderHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, sendPath) {
			mailjetHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, fixturesPath) {
			fixtureHandler.ServeHTTP(w, r)
			return
		}

		httperror.NotFound(w)
	})

	handler := server.ChainMiddlewares(mailerHandler, prometheusApp, opentracingApp, gzipApp, owaspApp, corsApp)

	serverApp.ListenAndServe(handler, nil, healthcheckApp)
}
