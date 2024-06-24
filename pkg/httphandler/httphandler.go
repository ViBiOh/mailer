package httphandler

import (
	"github.com/ViBiOh/mailer/pkg/mailer"
	"go.opentelemetry.io/otel/trace"
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
