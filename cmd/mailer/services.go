package main

import (
	"context"
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

type services struct {
	server      *server.Server
	amqpHandler *amqphandler.Service
	mailer      mailer.Service
}

func newServices(config configuration, clients clients) (services, error) {
	mjmlService := mjml.New(config.mjml, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	smtpService := smtp.New(config.smtp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	mailerService := mailer.New(config.mailer, mjmlService, smtpService, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	amqpHandler, err := amqphandler.New(config.amqphandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), mailerService.AmqpHandler)
	if err != nil {
		return services{}, fmt.Errorf("amqpHandler: %w", err)
	}

	return services{
		server:      server.New(config.server),
		mailer:      mailerService,
		amqpHandler: amqpHandler,
	}, nil
}

func (s services) Start(ctx context.Context) {
	go s.amqpHandler.Start(ctx)
}

func (s services) Close() {
	<-s.amqpHandler.Done()
}
