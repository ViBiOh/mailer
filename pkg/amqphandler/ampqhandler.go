package amqphandler

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/model"
)

// App of package
type App interface {
	Start(<-chan struct{})
	Ping() error
}

// Config of package
type Config struct {
	url      *string
	queue    *string
	exchange *string
	client   *string
}

type app struct {
	mailerApp mailer.App

	url      string
	exchange string
	queue    string
	client   string

	amqpClient model.AMQPClient
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:      flags.New(prefix, "amqp").Name("URL").Default("").Label("Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost>").ToString(fs),
		exchange: flags.New(prefix, "amqp").Name("Exchange").Default("mailer").Label("Exchange name").ToString(fs),
		queue:    flags.New(prefix, "amqp").Name("Queue").Default("mailer").Label("Queue name").ToString(fs),
		client:   flags.New(prefix, "amqp").Name("Name").Default("mailer").Label("Client name").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, mailerApp mailer.App) App {
	return &app{
		url:       strings.TrimSpace(*config.url),
		exchange:  strings.TrimSpace(*config.exchange),
		queue:     strings.TrimSpace(*config.queue),
		client:    strings.TrimSpace(*config.client),
		mailerApp: mailerApp,
	}
}

func (a *app) Start(done <-chan struct{}) {
	if len(a.url) == 0 {
		return
	}

	client, err := model.GetAMQPClient(a.url, a.exchange, a.queue, a.client)
	if err != nil {
		logger.Error("%s", err)
		return
	}
	defer client.Close()

	a.amqpClient = client

	messages, err := client.Listen()
	if err != nil {
		logger.Error("%s", err)
		return
	}

	logger.Info("Listening queue `%s` on vhost `%s`, on exchange `%s`", a.queue, client.Vhost(), a.exchange)

	for {
		select {
		case <-done:
			return
		case message := <-messages:
			if err := a.sendEmail(message.Body); err != nil {
				logger.Error("unable to send email: %s", err)

				if err := message.Reject(false); err != nil {
					logger.Error("unable to reject message: %s", err)
				}
			} else if err := message.Ack(false); err != nil {
				logger.Error("unable to ack message: %s", err)
			}

		}
	}
}

func (a *app) sendEmail(payload []byte) error {
	ctx := context.Background()

	var mailRequest model.MailRequest
	if err := json.Unmarshal(payload, &mailRequest); err != nil {
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	output, err := a.mailerApp.Render(ctx, mailRequest)
	if err != nil {
		return fmt.Errorf("unable to render email: %s", err)
	}

	return a.mailerApp.Send(ctx, mailRequest.ConvertToMail(output))
}

func (a *app) Ping() error {
	return a.amqpClient.Ping()
}
