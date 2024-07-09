package main

import (
	"context"
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/cors"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

type services struct {
	server *server.Server
	owasp  owasp.Service
	cors   cors.Service

	amqpHandler *amqphandler.Service
	mailer      mailer.Service
}

func newServices(config configuration, clients clients) (services, error) {
	var output services
	var err error

	output.server = server.New(config.server)

	if clients.amqp == nil && output.server == nil {
		return output, fmt.Errorf("no amqp or http listener")
	}

	output.owasp = owasp.New(config.owasp)
	output.cors = cors.New(config.cors)

	mjmlService := mjml.New(config.mjml, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	smtpService := smtp.New(config.smtp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	output.mailer = mailer.New(config.mailer, mjmlService, smtpService, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())

	output.amqpHandler, err = amqphandler.New(config.amqphandler, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.mailer.AmqpHandler)
	if err != nil {
		return output, fmt.Errorf("amqpHandler: %w", err)
	}

	return output, nil
}

func (s services) Start(ctx context.Context) {
	go s.amqpHandler.Start(ctx)
}

func (s services) Close() {
	<-s.amqpHandler.Done()
}
