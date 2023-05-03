package client

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/ViBiOh/flags"
	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

var (
	_ fmt.Stringer = App{}

	// ErrNotEnabled occurs when no configuration is provided
	ErrNotEnabled = errors.New("mailer not enabled")
)

// App of package
type App struct {
	amqpClient *amqpclient.Client
	tracer     trace.Tracer
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
		url:      flags.New("URL", "URL (https?:// or amqps?://)").Prefix(prefix).DocPrefix("mailer").String(fs, "", nil),
		name:     flags.New("Name", "HTTP Username or AMQP Exchange name").Prefix(prefix).DocPrefix("mailer").String(fs, "mailer", nil),
		password: flags.New("Password", "HTTP Pass").Prefix(prefix).DocPrefix("mailer").String(fs, "", nil),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegister prometheus.Registerer, tracer trace.Tracer) (App, error) {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return App{}, nil
	}

	name := strings.TrimSpace(*config.name)

	if strings.HasPrefix(url, "amqp") {
		client, err := amqpclient.NewFromURI(url, 1, prometheusRegister, tracer)
		if err != nil {
			return App{}, fmt.Errorf("create amqp client: %w", err)
		}

		if err := client.Publisher(name, "direct", nil); err != nil {
			return App{}, fmt.Errorf("configure amqp producer: %w", err)
		}

		return App{
			amqpClient: client,
			exchange:   name,
			tracer:     tracer,
		}, nil
	}

	return App{
		req: request.Post(url).BasicAuth(name, *config.password),
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
func (a App) Send(ctx context.Context, mailRequest model.MailRequest) (err error) {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "send")
	defer end(&err)

	if !a.Enabled() {
		return ErrNotEnabled
	}

	if err := mailRequest.Check(); err != nil {
		return err
	}

	if a.amqpEnabled() {
		return a.amqpClient.PublishJSON(ctx, mailRequest, a.exchange, "")
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
