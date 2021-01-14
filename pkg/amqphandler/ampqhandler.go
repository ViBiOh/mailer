package amqphandler

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/ViBiOh/httputils/v3/pkg/cron"
	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/model"
)

// App of package
type App interface {
	Start(<-chan struct{})
	Ping() error
	Close()
}

// Config of package
type Config struct {
	url      *string
	queue    *string
	exchange *string
	client   *string
}

type app struct {
	mailerApp  mailer.App
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
func New(config Config, mailerApp mailer.App) (App, error) {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return app{}, nil
	}

	client, err := model.GetAMQPClient(url, strings.TrimSpace(*config.exchange), strings.TrimSpace(*config.client), strings.TrimSpace(*config.queue))
	if err != nil {
		return nil, fmt.Errorf("unable to create amqp client: %s", err)
	}

	return app{
		mailerApp:  mailerApp,
		amqpClient: client,
	}, nil
}

func (a app) Start(done <-chan struct{}) {
	messages, err := a.amqpClient.Listen()
	if err != nil {
		logger.Error("unable to listen on queue: %s", err)
		return
	}

	logger.Info("Listening queue `%s` on vhost `%s`", a.amqpClient.QueueName(), a.amqpClient.Vhost())

	garbageCron := cron.New().Each(time.Hour).Now()
	defer garbageCron.Stop()

	go garbageCron.Start(func(_ time.Time) error {
		return a.garbageCollector(done)
	}, func(err error) {
		logger.Error("error while running garbage collector: %s", err)
	})

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

func (a app) garbageCollector(done <-chan struct{}) error {
	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for !isDone() {
		garbage, ok, err := a.amqpClient.GetGarbage()
		if err != nil {
			return fmt.Errorf("unable to get garbage message: %s", err)
		}

		if !ok {
			return nil
		}

		logger.Warn("garbage message: %s", garbage.Body)
		if err := garbage.Ack(false); err != nil {
			return fmt.Errorf("unable to ack garbage message: %s", err)
		}
	}

	return nil
}

func (a app) sendEmail(payload []byte) error {
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

func (a app) Ping() error {
	return a.amqpClient.Ping()
}

func (a app) Close() {
	a.amqpClient.Close()
}
