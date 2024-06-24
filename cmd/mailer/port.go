package main

import (
	"net/http"

	"github.com/ViBiOh/mailer/pkg/httphandler"
)

func newPort(clients clients, services services) http.Handler {
	mux := http.NewServeMux()

	handler := httphandler.New(services.mailer, clients.telemetry.TracerProvider())

	mux.HandleFunc("GET /fixtures/{fixture...}", handler.HandleFixture)
	mux.HandleFunc("GET /render/{template...}", handler.HandlerTemplate)
	mux.HandleFunc("GET /", handler.HandleRoot)

	return mux
}
