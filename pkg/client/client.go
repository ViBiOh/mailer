package client

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"

	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
)

var (
	_ fmt.Stringer = App{}

	// ErrNotEnabled occurs when no configuration is provided
	ErrNotEnabled = errors.New("mailer not enabled")
)

// App of package
type App struct {
	amqpClient *amqpclient.Client
	exchange   string
	req        request.Request
}

// Config of package
type Config struct {
	url      *string
	name     *string
	password *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:      flags.New(prefix, "mailer", "URL").Default("", nil).Label("URL (https?:// or amqps?://)").ToString(fs),
		name:     flags.New(prefix, "mailer", "Name").Default("mailer", nil).Label("HTTP Username or AMQP Exchange name").ToString(fs),
		password: flags.New(prefix, "mailer", "Password").Default("", nil).Label("HTTP Pass").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegister prometheus.Registerer) (App, error) {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return App{}, nil
	}

	name := strings.TrimSpace(*config.name)

	if strings.HasPrefix(url, "amqp") {
		client, err := amqpclient.NewFromURI(url, prometheusRegister)
		if err != nil {
			return App{}, fmt.Errorf("unable to create amqp client: %s", err)
		}

		if err := client.Publisher(name, "direct", nil); err != nil {
			return App{}, fmt.Errorf("unable to configure amqp producer: %s", err)
		}

		return App{
			amqpClient: client,
			exchange:   name,
		}, nil
	}

	return App{
		req: request.New().Post(url).BasicAuth(name, *config.password),
	}, nil
}

func (a App) String() string {
	if !a.Enabled() {
		return "not enabled"
	}

	if a.amqpEnabled() {
		return fmt.Sprintf("Publishing emails to exchange `%s` on vhost `%s`", a.exchange, a.amqpClient.Vhost())
	}
	return fmt.Sprintf("Sending emails via HTTP to `%v`.", a.req)
}

// Enabled checks if requirements are met
func (a App) Enabled() bool {
	return !a.req.IsZero() || a.amqpEnabled()
}

func (a App) amqpEnabled() bool {
	return a.amqpClient != nil && a.amqpClient.Enabled()
}

// Send sends emails with Mailer for defined parameters
func (a App) Send(ctx context.Context, mailRequest model.MailRequest) error {
	if !a.Enabled() {
		return ErrNotEnabled
	}

	if err := mailRequest.Check(); err != nil {
		return err
	}

	if a.amqpEnabled() {
		return a.amqpSend(ctx, mailRequest)
	}
	return a.httpSend(ctx, mailRequest)
}

// Close client
func (a App) Close() {
	if a.amqpEnabled() {
		a.amqpClient.Close()
	}
}

func (a App) httpSend(ctx context.Context, mail model.MailRequest) error {
	query := url.Values{
		"from":    []string{mail.FromEmail},
		"sender":  []string{mail.Sender},
		"subject": []string{mail.Subject},
		"to":      mail.Recipients,
	}

	queryPath := fmt.Sprintf("/render/%s?%s", url.PathEscape(mail.Tpl), query.Encode())

	_, err := a.req.Path(queryPath).JSON(ctx, mail.Payload)
	return err
}

func (a App) amqpSend(ctx context.Context, mailRequest model.MailRequest) error {
	payload, err := json.Marshal(mailRequest)
	if err != nil {
		return fmt.Errorf("unable to marshal mail: %s", err)
	}

	return a.amqpClient.Publish(amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
	}, a.exchange)
}
