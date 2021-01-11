package client

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/request"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/streadway/amqp"
)

var (
	// ErrNotEnabled occurs when no configuration is provided
	ErrNotEnabled = errors.New("mailer not enabled")
)

// App of package
type App interface {
	Enabled() bool
	Send(context.Context, model.MailRequest) error
	Close()
}

// Config of package
type Config struct {
	url      *string
	name     *string
	password *string
}

type app struct {
	url      string
	name     string
	password string

	amqpConnection *amqp.Connection
	amqpChannel    *amqp.Channel
	amqpQueue      amqp.Queue
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:      flags.New(prefix, "mailer").Name("URL").Default("").Label("URL (https?:// or amqps?://)").ToString(fs),
		name:     flags.New(prefix, "mailer").Name("Name").Default("mailer").Label("HTTP Username or AMQP Queue name").ToString(fs),
		password: flags.New(prefix, "mailer").Name("Password").Default("").Label("HTTP Pass").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) (App, error) {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return &app{}, nil
	}

	app := &app{}
	name := strings.TrimSpace(*config.name)

	if strings.HasPrefix(url, "amqp") {
		var err error
		app.amqpConnection, app.amqpChannel, app.amqpQueue, err = model.InitAMQP(url, name)
		if err != nil {
			app.Close()
			return nil, err
		}

		return app, nil
	}

	app.url = url
	app.name = name
	app.password = strings.TrimSpace(*config.password)
	return app, nil
}

func (a app) Enabled() bool {
	return len(a.url) != 0 || a.amqpConnection != nil
}

// Send sends emails with Mailer for defined parameters
func (a app) Send(ctx context.Context, mailRequest model.MailRequest) error {
	if !a.Enabled() {
		return ErrNotEnabled
	}

	if err := mailRequest.Check(); err != nil {
		return err
	}

	if a.amqpConnection != nil {
		return a.amqpSend(ctx, mailRequest)
	}
	return a.httpSend(ctx, mailRequest)
}

func (a app) Close() {
	if a.amqpChannel != nil {
		model.LoggedCloser(a.amqpChannel)
	}
	if a.amqpConnection != nil {
		model.LoggedCloser(a.amqpConnection)
	}
}

func (a app) httpSend(ctx context.Context, mail model.MailRequest) error {
	recipients := strings.Join(mail.Recipients, ",")

	url := fmt.Sprintf("%s/render/%s?from=%s&sender=%s&to=%s&subject=%s", a.url, url.QueryEscape(mail.Tpl), url.QueryEscape(mail.FromEmail), url.QueryEscape(mail.Sender), url.QueryEscape(recipients), url.QueryEscape(mail.Subject))

	req := request.New().Post(url)
	if a.password != "" {
		req.BasicAuth(a.name, a.password)
	}

	_, err := req.JSON(ctx, mail.Payload)
	return err
}

func (a app) amqpSend(ctx context.Context, mailRequest model.MailRequest) error {
	payload, err := json.Marshal(mailRequest)
	if err != nil {
		return fmt.Errorf("unable to marshal mail: %s", err)
	}

	if err := a.amqpChannel.Publish("", a.amqpQueue.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
	}); err != nil {
		return fmt.Errorf("unable to publish message: %s", err)
	}

	return nil
}
