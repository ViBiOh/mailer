package amqphandler

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/mailer/pkg/mailer"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/streadway/amqp"
)

// App of package
type App interface {
	Start(<-chan struct{})
	Ping() error
}

// Config of package
type Config struct {
	url *string
}

type app struct {
	amqpConnection *amqp.Connection
	mailerApp      mailer.App
	url            string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url: flags.New(prefix, "amqp").Name("URL").Default("").Label("Address").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, mailerApp mailer.App) App {
	return &app{
		url:       strings.TrimSpace(*config.url),
		mailerApp: mailerApp,
	}
}

func (a *app) Start(done <-chan struct{}) {
	if len(a.url) == 0 {
		return
	}

	conn, channel, queue, err := model.InitAMQP(a.url)
	if conn != nil {
		defer model.LoggedCloser(conn)
		a.amqpConnection = conn
	}
	if channel != nil {
		defer model.LoggedCloser(channel)
	}
	if err != nil {
		logger.Error("%s", err)
		return
	}

	messages, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		logger.Error("unable to consume queue `%s`: %s", queue.Name, err)
		return
	}

	logger.Info("Consuming queue `%s` on vhost `%s`", queue.Name, conn.Config.Vhost)

	for {
		select {
		case <-done:
			return
		case message := <-messages:
			if err := a.sendEmail(message.Body); err != nil {
				logger.Error("unable to send email: %s", err)
			}
			if err := message.Ack(true); err != nil {
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

	if err := a.mailerApp.Send(ctx, mailRequest.ConvertToMail(output)); err != nil {
		return fmt.Errorf("unable to send email: %s", err)
	}

	return nil
}

func (a *app) Ping() error {
	if a.amqpConnection == nil {
		return nil
	}

	if a.amqpConnection.IsClosed() {
		return errors.New("amqp closed")
	}

	return nil
}
