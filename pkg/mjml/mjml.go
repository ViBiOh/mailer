package mjml

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/mailer/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
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
		url:      flags.String(fs, prefix, "mjml", "URL", "MJML API Converter URL", "https://api.mjml.io/v1/render", nil),
		username: flags.String(fs, prefix, "mjml", "Username", "Application ID or Basic Auth username", "", nil),
		password: flags.String(fs, prefix, "mjml", "Password", "Secret Key or Basic Auth password", "", nil),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegisterer prometheus.Registerer, tracer trace.Tracer) App {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return App{}
	}

	metric.Create(prometheusRegisterer, "mjml")

	return App{
		req:    request.Post(url).BasicAuth(strings.TrimSpace(*config.username), *config.password).WithClient(request.CreateClient(time.Minute, request.NoRedirection)),
		tracer: tracer,
	}
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

	ctx, end := tracer.StartSpan(ctx, a.tracer, "render")
	defer end(&err)

	resp, err := a.req.JSON(ctx, mjmlRequest{template})
	if err != nil {
		metric.Increase("mjml", "error")
		return "", fmt.Errorf("render mjml template: %w", err)
	}

	var response mjmlResponse
	if err = httpjson.Read(resp, &response); err != nil {
		return "", fmt.Errorf("read mjml response: %w", err)
	}

	metric.Increase("mjml", "success")

	return response.HTML, nil
}
