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

type Service struct {
	tracer        trace.Tracer
	mailerService mailer.Service
}

func New(mailerService mailer.Service, tracerProvider trace.TracerProvider) Service {
	service := Service{
		mailerService: mailerService,
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("mailer_http")
	}

	return service
}

func (s Service) Handler() http.Handler {
	renderHandler := http.StripPrefix(renderPath, s.renderHandler())
	fixtureHandler := http.StripPrefix(fixturesPath, s.fixturesHandler())

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
