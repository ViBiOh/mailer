package main

import (
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
	"github.com/ViBiOh/mailer/healthcheck"
	"github.com/ViBiOh/mailer/mailjet"
	"github.com/ViBiOh/mailer/render"
)

const (
	healthcheckPath = `/health`
	mailPath        = `/mail`
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
	mailjetConfig := mailjet.Flags(``)
	authConfig := auth.Flags(`auth`)
	basicConfig := basic.Flags(`basic`)

	httputils.StartMainServer(func() http.Handler {
		mailjetApp := mailjet.NewApp(mailjetConfig)

		renderApp := render.NewApp()
		renderHandler := http.StripPrefix(mailPath, renderApp.Handler())

		healthcheckApp := healthcheck.NewApp(mailjetApp)
		healthcheckHandler := http.StripPrefix(healthcheckPath, healthcheckApp.Handler())

		authApp := auth.NewApp(authConfig, authService.NewBasicApp(basicConfig))
		authHandler := authApp.HandlerWithFail(func(w http.ResponseWriter, r *http.Request, _ *authProvider.User) {
			if strings.HasPrefix(r.URL.Path, mailPath) {
				renderHandler.ServeHTTP(w, r)
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
	}, nil)
}
