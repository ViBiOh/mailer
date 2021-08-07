package mjml

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

var (
	prefix = []byte("<mjml>")
)

type mjmlRequest struct {
	Mjml string `json:"mjml"`
}

type mjmlResponse struct {
	HTML string `json:"html"`
	Mjml string `json:"mjml"`
}

// App of package
type App struct {
	url      string
	username string
	password string
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
		url:      flags.New(prefix, "mjml").Name("URL").Default("https://api.mjml.io/v1/render").Label("MJML API Converter URL").ToString(fs),
		username: flags.New(prefix, "mjml").Name("Username").Default("").Label("Application ID or Basic Auth username").ToString(fs),
		password: flags.New(prefix, "mjml").Name("Password").Default("").Label("Secret Key or Basic Auth password").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	converter := strings.TrimSpace(*config.url)

	if converter == "" {
		return App{}
	}

	app := App{
		url:      converter,
		username: strings.TrimSpace(*config.username),
		password: strings.TrimSpace(*config.password),
	}

	return app
}

// IsMJML determines if provided content is a MJML template or not
func IsMJML(content []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(content), prefix)
}

// Enabled checks if requirements are met
func (a App) Enabled() bool {
	return a.url != ""
}

// Render MJML template
func (a App) Render(ctx context.Context, template string) (string, error) {
	if !a.Enabled() {
		return template, nil
	}

	resp, err := request.New().Post(a.url).BasicAuth(a.username, a.password).JSON(ctx, mjmlRequest{template})
	if err != nil {
		return "", fmt.Errorf("unable to render mjml template: %s", err)
	}

	var response mjmlResponse
	if err := httpjson.Read(resp, &response); err != nil {
		return "", fmt.Errorf("unable to read mjml response: %s", err)
	}

	return response.HTML, nil
}
