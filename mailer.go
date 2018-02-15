package main

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cors"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/mailer/healthcheck"
	"github.com/ViBiOh/mailer/mailjet"
	"github.com/ViBiOh/mailer/render"
)

const (
	healthcheckPath = `/health`
	mailPath        = `/mail`
)

func main() {
	corsConfig := cors.Flags(`cors`)
	owaspConfig := owasp.Flags(``)
	mailjetConfig := mailjet.Flags(``)

	httputils.StartMainServer(func() http.Handler {
		mailjetApp := mailjet.NewApp(mailjetConfig)

		renderApp := render.NewApp()
		renderHandler := http.StripPrefix(mailPath, renderApp.Handler())

		healthcheckApp := healthcheck.NewApp(mailjetApp)
		healthcheckHandler := http.StripPrefix(healthcheckPath, healthcheckApp.Handler())

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, healthcheckPath) {
				healthcheckHandler.ServeHTTP(w, r)
			} else if strings.HasPrefix(r.URL.Path, mailPath) {
				renderHandler.ServeHTTP(w, r)
			} else {
				httputils.NotFound(w)
			}
		})

		return owasp.Handler(owaspConfig, cors.Handler(corsConfig, handler))
	})
}
