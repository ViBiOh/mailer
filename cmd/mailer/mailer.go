package main

import (
	"context"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/cors"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

func main() {
	config := newConfig()
	alcotest.DoAndExit(config.alcotest)

	ctx := context.Background()

	clients, err := newClients(ctx, config)
	logger.FatalfOnErr(ctx, err, "client")

	go clients.Start()
	defer clients.Close(ctx)

	services, err := newServices(config, clients)
	logger.FatalfOnErr(ctx, err, "services")

	go services.Start(clients.health.EndCtx())
	defer services.Close()

	port := newPort(clients, services)

	go services.server.Start(clients.health.EndCtx(), httputils.Handler(port, clients.health, clients.telemetry.Middleware("http"), owasp.New(config.owasp).Middleware, cors.New(config.cors).Middleware))

	clients.health.WaitForTermination(getDoneChan(services.server.Done(), clients.amqp, services.amqpHandler))

	server.GracefulWait(services.server.Done(), services.amqpHandler.Done())
}

func getDoneChan(httpDone <-chan struct{}, amqpClient *amqp.Client, amqpService *amqphandler.Service) <-chan struct{} {
	if amqpClient == nil {
		return httpDone
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		select {
		case <-httpDone:
		case <-amqpService.Done():
		}
	}()

	return done
}
