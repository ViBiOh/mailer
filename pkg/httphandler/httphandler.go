package httphandler

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/mailer/pkg/mailer"
)

const (
	renderPath   = "/render"
	fixturesPath = "/fixtures"
)

// App of package
type App struct {
	mailerApp mailer.App
}

// New creates new App
func New(mailerApp mailer.App) App {
	return App{
		mailerApp: mailerApp,
	}
}

// Handler for Render request. Should be use with net/http
func (a App) Handler() http.Handler {
	renderHandler := http.StripPrefix(renderPath, a.renderHandler())
	fixtureHandler := http.StripPrefix(fixturesPath, a.fixturesHandler())

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, renderPath) {
			renderHandler.ServeHTTP(w, r)
			return
		}

		if strings.HasPrefix(r.URL.Path, fixturesPath) {
			fixtureHandler.ServeHTTP(w, r)
			return
		}

		httperror.NotFound(w)
	})
}
