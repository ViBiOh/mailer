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
	"github.com/ViBiOh/mailer/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
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
	req request.Request
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
		url:      flags.New(prefix, "mjml", "URL").Default("https://api.mjml.io/v1/render", nil).Label("MJML API Converter URL").ToString(fs),
		username: flags.New(prefix, "mjml", "Username").Default("", nil).Label("Application ID or Basic Auth username").ToString(fs),
		password: flags.New(prefix, "mjml", "Password").Default("", nil).Label("Secret Key or Basic Auth password").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegisterer prometheus.Registerer) App {
	url := strings.TrimSpace(*config.url)
	if len(url) == 0 {
		return App{}
	}

	metric.Create(prometheusRegisterer, "mjml")

	return App{
		req: request.Post(url).BasicAuth(strings.TrimSpace(*config.username), *config.password),
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

	resp, err := a.req.JSON(ctx, mjmlRequest{template})
	if err != nil {
		metric.Increase("mjml", "error")
		return "", fmt.Errorf("unable to render mjml template: %s", err)
	}

	var response mjmlResponse
	if err := httpjson.Read(resp, &response); err != nil {
		return "", fmt.Errorf("unable to read mjml response: %s", err)
	}

	metric.Increase("mjml", "success")

	return response.HTML, nil
}
