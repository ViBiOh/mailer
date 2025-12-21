package mjml

import (
	"bytes"
	"context"
	"flag"
	"fmt"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	mailer_metric "github.com/ViBiOh/mailer/pkg/metric"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var prefix = []byte("<mjml>")

type mjmlRequest struct {
	Mjml string `json:"mjml"`
}

type mjmlResponse struct {
	HTML string `json:"html"`
	Mjml string `json:"mjml"`
}

type Service struct {
	tracer trace.Tracer
	req    request.Request
}

type Config struct {
	URL      string
	Username string
	Password string
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("URL", "MJML API Converter URL").Prefix(prefix).DocPrefix("mjml").StringVar(fs, &config.URL, "https://api.mjml.io/v1/render", nil)
	flags.New("Username", "Application ID or Basic Auth username").Prefix(prefix).DocPrefix("mjml").StringVar(fs, &config.Username, "", nil)
	flags.New("Password", "Secret Key or Basic Auth password").Prefix(prefix).DocPrefix("mjml").StringVar(fs, &config.Password, "", nil)

	return &config
}

func New(config *Config, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) Service {
	if len(config.URL) == 0 {
		return Service{}
	}

	mailer_metric.Create(meterProvider, "mailer.mjml")

	fmt.Println(config.URL)

	service := Service{
		req: request.Post(config.URL).BasicAuth(config.Username, config.Password),
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("mjml")
	}

	return service
}

func IsMJML(content []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(content), prefix)
}

func (s Service) Enabled() bool {
	return !s.req.IsZero()
}

func (s Service) Render(ctx context.Context, template string) (string, error) {
	if !s.Enabled() {
		return template, nil
	}

	var err error

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "render")
	defer end(&err)

	resp, err := s.req.JSON(ctx, mjmlRequest{template})
	if err != nil {
		mailer_metric.Increase(ctx, "mjml", "error")
		return "", fmt.Errorf("render mjml template: %w", err)
	}

	response, err := httpjson.Read[mjmlResponse](resp)
	if err != nil {
		return "", fmt.Errorf("read mjml response: %w", err)
	}

	mailer_metric.Increase(ctx, "mjml", "success")

	return response.HTML, nil
}
