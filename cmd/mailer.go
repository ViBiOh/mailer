package main

import (
	"flag"
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/pkg/auth"
	"github.com/ViBiOh/auth/pkg/model"
	authProvider "github.com/ViBiOh/auth/pkg/provider"
	"github.com/ViBiOh/auth/pkg/provider/basic"
	authService "github.com/ViBiOh/auth/pkg/service"
	"github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/alcotest"
	"github.com/ViBiOh/httputils/pkg/cors"
	"github.com/ViBiOh/httputils/pkg/gzip"
	"github.com/ViBiOh/httputils/pkg/healthcheck"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/opentracing"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/httputils/pkg/server"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	mailerHealthcheck "github.com/ViBiOh/mailer/pkg/healthcheck"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/render"
)

const (
	fixturesPath = `/fixtures`
	renderPath   = `/render`
	sendPath     = `/send`
)

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error) {
	if auth.IsForbiddenErr(err) {
		httperror.Forbidden(w)
	} else if err == authProvider.ErrMalformedAuth || err == authProvider.ErrUnknownAuthType {
		httperror.BadRequest(w, err)
	} else {
		w.Header().Add(`WWW-Authenticate`, `Basic charset="UTF-8"`)
		httperror.Unauthorized(w, err)
	}
}

func main() {
	serverConfig := httputils.Flags(``)
	alcotestConfig := alcotest.Flags(``)
	opentracingConfig := opentracing.Flags(`tracing`)
	owaspConfig := owasp.Flags(``)
	corsConfig := cors.Flags(`cors`)

	mailjetConfig := mailjet.Flags(`mailjet`)
	mjmlConfig := mjml.Flags(`mjml`)
	authConfig := auth.Flags(`auth`)
	basicConfig := basic.Flags(`basic`)

	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	serverApp := httputils.NewApp(serverConfig)
	healthcheckApp := healthcheck.NewApp()
	opentracingApp := opentracing.NewApp(opentracingConfig)
	owaspApp := owasp.NewApp(owaspConfig)
	corsApp := cors.NewApp(corsConfig)
	gzipApp := gzip.NewApp()

	mjmlApp := mjml.NewApp(mjmlConfig)
	mailjetApp := mailjet.NewApp(mailjetConfig)
	renderApp := render.NewApp(mjmlApp, mailjetApp)

	healthcheckApp.NextHealthcheck(mailerHealthcheck.NewApp(mailjetApp).Handler())

	mailjetHandler := mailjetApp.Handler()
	renderHandler := http.StripPrefix(renderPath, renderApp.Handler())
	fixtureHandler := http.StripPrefix(fixturesPath, fixtures.Handler())

	authApp := auth.NewApp(authConfig, authService.NewBasicApp(basicConfig))
	authHandler := authApp.HandlerWithFail(func(w http.ResponseWriter, r *http.Request, _ *model.User) {
		if strings.HasPrefix(r.URL.Path, renderPath) {
			renderHandler.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, sendPath) {
			mailjetHandler.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, fixturesPath) {
			fixtureHandler.ServeHTTP(w, r)
		} else {
			httperror.NotFound(w)
		}
	}, handleAnonymousRequest)

	handler := server.ChainMiddlewares(authHandler, opentracingApp, gzipApp, owaspApp, corsApp)

	serverApp.ListenAndServe(handler, nil, healthcheckApp)
}
