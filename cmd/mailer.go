package main

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/pkg/auth"
	authProvider "github.com/ViBiOh/auth/pkg/provider"
	"github.com/ViBiOh/auth/pkg/provider/basic"
	authService "github.com/ViBiOh/auth/pkg/service"
	"github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/cors"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/healthcheck"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/render"
)

const (
	healthcheckPath = `/health`
	fixturesPath    = `/fixtures`
	renderPath      = `/render`
)

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error) {
	if auth.IsForbiddenErr(err) {
		httperror.Forbidden(w)
	} else if err == authProvider.ErrMalformedAuth || err == authProvider.ErrUnknownAuthType {
		httperror.BadRequest(w, err)
	} else {
		w.Header().Add(`WWW-Authenticate`, `Basic`)
		httperror.Unauthorized(w, err)
	}
}

func main() {
	corsConfig := cors.Flags(`cors`)
	owaspConfig := owasp.Flags(``)
	mailjetConfig := mailjet.Flags(`mailjet`)
	mjmlConfig := mjml.Flags(`mjml`)
	authConfig := auth.Flags(`auth`)
	basicConfig := basic.Flags(`basic`)

	httputils.NewApp(httputils.Flags(``), func() http.Handler {
		mailjetApp := mailjet.NewApp(mailjetConfig)
		mjmlApp := mjml.NewApp(mjmlConfig)

		renderApp := render.NewApp(mjmlApp)
		renderHandler := http.StripPrefix(renderPath, renderApp.Handler())

		fixtureHandler := http.StripPrefix(fixturesPath, fixtures.Handler())

		healthcheckApp := healthcheck.NewApp(mailjetApp)
		healthcheckHandler := http.StripPrefix(healthcheckPath, healthcheckApp.Handler())

		authApp := auth.NewApp(authConfig, authService.NewBasicApp(basicConfig))
		authHandler := authApp.HandlerWithFail(func(w http.ResponseWriter, r *http.Request, _ *authProvider.User) {
			if strings.HasPrefix(r.URL.Path, renderPath) {
				renderHandler.ServeHTTP(w, r)
			} else if strings.HasPrefix(r.URL.Path, fixturesPath) {
				fixtureHandler.ServeHTTP(w, r)
			} else {
				httperror.NotFound(w)
			}
		}, handleAnonymousRequest)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, healthcheckPath) {
				healthcheckHandler.ServeHTTP(w, r)
			} else {
				authHandler.ServeHTTP(w, r)
			}
		})

		return owasp.Handler(owaspConfig, cors.Handler(corsConfig, handler))
	}, nil).ListenAndServe()
}
