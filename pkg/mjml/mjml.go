package mjml

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/request"
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
type App interface {
	Render(context.Context, string) (string, error)
}

// Config of package
type Config struct {
	url      *string
	username *string
	password *string
}

type app struct {
	url      string
	username string
	password string
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
		return app{}
	}

	app := app{
		url:      converter,
		username: strings.TrimSpace(*config.username),
		password: strings.TrimSpace(*config.password),
	}

	return app
}

// IsMJML determines if provided content is a MJML template or not
func IsMJML(content []byte) bool {
	return bytes.HasPrefix(content, prefix)
}

func (a app) isReady() bool {
	return a.url != ""
}

// Render MJML template
func (a app) Render(ctx context.Context, template string) (string, error) {
	if !a.isReady() {
		return template, nil
	}

	resp, err := request.New().Post(a.url).BasicAuth(a.username, a.password).JSON(ctx, mjmlRequest{template})
	if err != nil {
		return "", fmt.Errorf("unable to render mjml template: %s", err)
	}

	content, err := request.ReadBodyResponse(resp)
	if err != nil {
		return "", fmt.Errorf("unable to read mjml response: %s", err)
	}

	var response mjmlResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return "", fmt.Errorf("unable to parse mjml response: %s", err)
	}

	return response.HTML, nil
}
