package mjml

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"strings"

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

// App of package
type App struct {
	tracer trace.Tracer
	req    request.Request
}

// Config of package
type Config struct {
	url      *string
	username *string
	password *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:      flags.New("URL", "MJML API Converter URL").Prefix(prefix).DocPrefix("mjml").String(fs, "https://api.mjml.io/v1/render", nil),
		username: flags.New("Username", "Application ID or Basic Auth username").Prefix(prefix).DocPrefix("mjml").String(fs, "", nil),
		password: flags.New("Password", "Secret Key or Basic Auth password").Prefix(prefix).DocPrefix("mjml").String(fs, "", nil),
	}
}

// New creates new App from Config
func New(config Config, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) App {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return App{}
	}

	mailer_metric.Create(meterProvider, "mailer.mjml")

	app := App{
		req: request.Post(url).BasicAuth(strings.TrimSpace(*config.username), *config.password),
	}

	if tracerProvider != nil {
		app.tracer = tracerProvider.Tracer("mjml")
	}

	return app
}

// IsMJML determines if provided content is a MJML template or not
func IsMJML(content []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(content), prefix)
}

// Enabled checks if requirements are met
func (a App) Enabled() bool {
	return !a.req.IsZero()
}

// Render MJML template
func (a App) Render(ctx context.Context, template string) (string, error) {
	if !a.Enabled() {
		return template, nil
	}

	var err error

	ctx, end := telemetry.StartSpan(ctx, a.tracer, "render")
	defer end(&err)

	resp, err := a.req.JSON(ctx, mjmlRequest{template})
	if err != nil {
		mailer_metric.Increase(ctx, "mjml", "error")
		return "", fmt.Errorf("render mjml template: %w", err)
	}

	var response mjmlResponse
	if err = httpjson.Read(resp, &response); err != nil {
		return "", fmt.Errorf("read mjml response: %w", err)
	}

	mailer_metric.Increase(ctx, "mjml", "success")

	return response.HTML, nil
}
