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
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/mailer/pkg/model"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	_ fmt.Stringer = Service{}

	// ErrNotEnabled occurs when no configuration is provided
	ErrNotEnabled = errors.New("mailer not enabled")
)

type Service struct {
	amqpClient *amqpclient.Client
	tracer     trace.Tracer
	exchange   string
	req        request.Request
}

type Config struct {
	URL      string
	Name     string
	Password string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	var config Config

	flags.New("URL", "URL (https?:// or amqps?://)").Prefix(prefix).DocPrefix("mailer").StringVar(fs, &config.URL, "", nil)
	flags.New("Name", "HTTP Username or AMQP Exchange name").Prefix(prefix).DocPrefix("mailer").StringVar(fs, &config.Name, "mailer", nil)
	flags.New("Password", "HTTP Pass").Prefix(prefix).DocPrefix("mailer").StringVar(fs, &config.Password, "", nil)

	return config
}

func New(config Config, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) (Service, error) {
	if len(config.URL) == 0 {
		return Service{}, nil
	}

	if strings.HasPrefix(config.URL, "amqp") {
		client, err := amqpclient.NewFromURI(config.URL, 1, meterProvider, tracerProvider)
		if err != nil {
			return Service{}, fmt.Errorf("create amqp client: %w", err)
		}

		if err := client.Publisher(config.Name, "direct", nil); err != nil {
			return Service{}, fmt.Errorf("configure amqp producer: %w", err)
		}

		service := Service{
			amqpClient: client,
			exchange:   config.Name,
		}

		if tracerProvider != nil {
			service.tracer = tracerProvider.Tracer("mailer")
		}

		return service, nil
	}

	return Service{
		req: request.Post(config.URL).BasicAuth(config.Name, config.Password),
	}, nil
}

func (s Service) String() string {
	if !s.Enabled() {
		return "not enabled"
	}

	if s.amqpEnabled() {
		return fmt.Sprintf("Publishing emails to exchange `%s` on vhost `%s`", s.exchange, s.amqpClient.Vhost())
	}
	return fmt.Sprintf("Sending emails via HTTP to `%v`.", s.req)
}

func (s Service) Enabled() bool {
	return !s.req.IsZero() || s.amqpEnabled()
}

func (s Service) amqpEnabled() bool {
	return s.amqpClient != nil && s.amqpClient.Enabled()
}

func (s Service) Send(ctx context.Context, mailRequest model.MailRequest) (err error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "send")
	defer end(&err)

	if !s.Enabled() {
		return ErrNotEnabled
	}

	if err := mailRequest.Check(); err != nil {
		return err
	}

	if s.amqpEnabled() {
		return s.amqpClient.PublishJSON(ctx, mailRequest, s.exchange, "")
	}

	return s.httpSend(ctx, mailRequest)
}

func (s Service) Close() {
	if s.amqpEnabled() {
		s.amqpClient.Close()
	}
}

func (s Service) httpSend(ctx context.Context, mail model.MailRequest) error {
	query := url.Values{
		"from":    []string{mail.FromEmail},
		"sender":  []string{mail.Sender},
		"subject": []string{mail.Subject},
		"to":      mail.Recipients,
	}

	queryPath := fmt.Sprintf("/render/%s?%s", url.PathEscape(mail.Tpl), query.Encode())

	_, err := s.req.Path(queryPath).JSON(ctx, mail.Payload)
	return err
}
