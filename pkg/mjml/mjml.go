package mjml

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v2/pkg/errors"
	"github.com/ViBiOh/httputils/v2/pkg/request"
	"github.com/ViBiOh/httputils/v2/pkg/tools"
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

// Config of package
type Config struct {
	url  *string
	user *string
	pass *string
}

// App of package
type App struct {
	url     string
	headers http.Header
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:  tools.NewFlag(prefix, "mjml").Name("URL").Default("https://api.mjml.io/v1/render").Label("MJML API Converter URL").ToString(fs),
		user: tools.NewFlag(prefix, "mjml").Name("User").Default("").Label("Application ID or Basic Auth user").ToString(fs),
		pass: tools.NewFlag(prefix, "mjml").Name("Pass").Default("").Label("Secret Key or Basic Auth pass").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) *App {
	converter := strings.TrimSpace(*config.url)
	user := strings.TrimSpace(*config.user)
	pass := strings.TrimSpace(*config.pass)

	if converter == "" {
		return &App{}
	}

	app := App{
		url: converter,
	}

	if user != "" && pass != "" {
		app.headers = http.Header{"Authorization": []string{request.GenerateBasicAuth(user, pass)}}
	}

	return &app
}

// IsMJML determines if provided content is a MJML template or not
func IsMJML(content []byte) bool {
	return bytes.HasPrefix(content, prefix)
}

func (a App) isReady() bool {
	return a.url != ""
}

// Render MJML template
func (a App) Render(ctx context.Context, template string) (string, error) {
	if !a.isReady() {
		return template, nil
	}

	body, _, _, err := request.DoJSON(ctx, a.url, mjmlRequest{template}, a.headers, http.MethodPost)
	if err != nil {
		return "", err
	}

	content, err := request.ReadBody(body)
	if err != nil {
		return "", err
	}

	var response mjmlResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return "", errors.WithStack(err)
	}

	return response.HTML, nil
}
