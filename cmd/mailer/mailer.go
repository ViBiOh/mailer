package main

import (
	"context"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
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

	go services.server.Start(clients.health.EndCtx(), port)

	clients.health.WaitForTermination(getDoneChan(services.server, clients.amqp, services.amqpHandler))
	health.WaitAll(services.server.Done(), services.amqpHandler.Done())
}

func getDoneChan(httpServer *server.Server, amqpClient *amqp.Client, amqpService *amqphandler.Service) <-chan struct{} {
	var httpDone <-chan struct{}
	if httpServer != nil {
		httpDone = httpServer.Done()
	}

	var amqpDone <-chan struct{}
	if amqpClient != nil {
		amqpDone = amqpService.Done()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		select {
		case <-httpDone:
		case <-amqpDone:
		}
	}()

	return done
}
