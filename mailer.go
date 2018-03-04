package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/auth"
	authProvider "github.com/ViBiOh/auth/provider"
	"github.com/ViBiOh/auth/provider/basic"
	authService "github.com/ViBiOh/auth/service"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cors"
	"github.com/ViBiOh/httputils/httperror"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/mailer/fixtures"
	"github.com/ViBiOh/mailer/healthcheck"
	"github.com/ViBiOh/mailer/mailjet"
	"github.com/ViBiOh/mailer/mjml"
	"github.com/ViBiOh/mailer/render"
	"github.com/ViBiOh/viws/viws"
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
	viwsConfig := viws.Flags(``)

	httputils.StartMainServer(func() http.Handler {
		mailjetApp := mailjet.NewApp(mailjetConfig)
		mjmlApp := mjml.NewApp(mjmlConfig)

		renderApp := render.NewApp(mjmlApp)
		renderHandler := http.StripPrefix(renderPath, renderApp.Handler())

		fixtureHandler := http.StripPrefix(fixturesPath, fixtures.Handler())

		viwsApp, err := viws.NewApp(viwsConfig)
		if err != nil {
			log.Fatalf(`Error while initializing viws: %v`, err)
		}
		viwsHandler := viwsApp.FileHandler()

		healthcheckApp := healthcheck.NewApp(mailjetApp)
		healthcheckHandler := http.StripPrefix(healthcheckPath, healthcheckApp.Handler())

		authApp := auth.NewApp(authConfig, authService.NewBasicApp(basicConfig))
		authHandler := authApp.HandlerWithFail(func(w http.ResponseWriter, r *http.Request, _ *authProvider.User) {
			if strings.HasPrefix(r.URL.Path, renderPath) {
				renderHandler.ServeHTTP(w, r)
			} else if strings.HasPrefix(r.URL.Path, fixturesPath) {
				fixtureHandler.ServeHTTP(w, r)
			} else {
				viwsHandler.ServeHTTP(w, r)
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
	}, nil)
}
