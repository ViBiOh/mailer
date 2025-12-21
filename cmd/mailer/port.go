package main

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/mailer/pkg/httphandler"
)

func newPort(clients clients, services services) http.Handler {
	mux := http.NewServeMux()

	handler := httphandler.New(services.mailer, clients.telemetry.TracerProvider())

	mux.HandleFunc("GET /fixtures/{fixture...}", handler.HandleFixture)
	mux.HandleFunc("GET /render/{template...}", handler.HandlerTemplate)
	mux.HandleFunc("POST /render/{template...}", handler.HandlerTemplate)
	mux.HandleFunc("GET /", handler.HandleRoot)

	return httputils.Handler(mux, clients.health,
		clients.telemetry.Middleware("http"),
		services.owasp.Middleware,
		services.cors.Middleware,
	)
}
