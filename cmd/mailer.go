package main

import (
	"flag"
	"net/http"
	"os"
	"path"
	"strings"

	httputils "github.com/ViBiOh/httputils/v3/pkg"
	"github.com/ViBiOh/httputils/v3/pkg/alcotest"
	"github.com/ViBiOh/httputils/v3/pkg/cors"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/render"
)

const (
	fixturesPath = "/fixtures"
	renderPath   = "/render"

	docPath = "doc/"
)

func main() {
	fs := flag.NewFlagSet("mailer", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")
	corsConfig := cors.Flags(fs, "cors")

	mailjetConfig := mailjet.Flags(fs, "mailjet")
	mjmlConfig := mjml.Flags(fs, "mjml")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	prometheusApp := prometheus.New(prometheusConfig)
	owaspApp := owasp.New(owaspConfig)
	corsApp := cors.New(corsConfig)

	mjmlApp := mjml.New(mjmlConfig)
	mailjetApp := mailjet.New(mailjetConfig)
	renderApp := render.New(mjmlApp, mailjetApp)

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

		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path.Join(docPath, r.URL.Path))
	})

	handler := httputils.ChainMiddlewares(mailerHandler, prometheusApp, owaspApp, corsApp)

	httputils.New(serverConfig).ListenAndServe(handler, httputils.HealthHandler(nil), nil)
}
