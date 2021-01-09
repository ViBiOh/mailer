package httphandler

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/query"
	"github.com/ViBiOh/mailer/pkg/mailer"
)

const (
	renderPath   = "/render"
	fixturesPath = "/fixtures"
)

// App of package
type App interface {
	Handler() http.Handler
}

type app struct {
	mailerApp mailer.App
}

// New creates new App
func New(mailerApp mailer.App) App {
	return app{
		mailerApp: mailerApp,
	}
}

// Handler for Render request. Should be use with net/http
func (a app) Handler() http.Handler {
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

func checkRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost:
		return !query.IsRoot(r)
	case http.MethodGet:
		return true
	default:
		return false
	}
}
