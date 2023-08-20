package httphandler

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"go.opentelemetry.io/otel/trace"
)

const (
	renderPath   = "/render"
	fixturesPath = "/fixtures"
)

// App of package
type App struct {
	tracer    trace.Tracer
	mailerApp mailer.App
}

// New creates new App
func New(mailerApp mailer.App, tracerProvider trace.TracerProvider) App {
	app := App{
		mailerApp: mailerApp,
	}

	if tracerProvider != nil {
		app.tracer = tracerProvider.Tracer("mailer_http")
	}

	return app
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
