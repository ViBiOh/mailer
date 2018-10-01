package mjml

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/tools"
)

var (
	prefix = []byte(`<mjml>`)
)

type mjmlRequest struct {
	Mjml string `json:"mjml"`
}

type mjmlResponse struct {
	HTML string `json:"html"`
	Mjml string `json:"mjml"`
}

// App stores informations
type App struct {
	url     string
	headers http.Header
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) *App {
	converter := strings.TrimSpace(*config[`url`])
	user := strings.TrimSpace(*config[`user`])
	pass := strings.TrimSpace(*config[`pass`])

	if converter == `` {
		return &App{}
	}

	app := App{
		url: converter,
	}

	if user != `` && pass != `` {
		app.headers = http.Header{`Authorization`: []string{request.GenerateBasicAuth(user, pass)}}
	}

	return &app
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`url`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sURL`, prefix)), `https://api.mjml.io/v1/render`, `[mjml] MJML API Converter URL`),
		`user`: flag.String(tools.ToCamel(fmt.Sprintf(`%sUser`, prefix)), ``, `[mjml] Application ID or Basic Auth user`),
		`pass`: flag.String(tools.ToCamel(fmt.Sprintf(`%sPass`, prefix)), ``, `[mjml] Secret Key or Basic Auth pass`),
	}
}

// IsMJML determines if provided content is a MJML template or not
func IsMJML(content []byte) bool {
	return bytes.HasPrefix(content, prefix)
}

func (a App) isReady() bool {
	return a.url != ``
}

// Render MJML template
func (a App) Render(ctx context.Context, template string) (string, error) {
	if !a.isReady() {
		return template, nil
	}

	content, err := request.DoJSON(ctx, a.url, mjmlRequest{template}, a.headers, http.MethodPost)
	if err != nil {
		return ``, fmt.Errorf(`error while sending data: %s: %s`, err, content)
	}

	var response mjmlResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return ``, fmt.Errorf(`error while unmarshalling data: %v: %s`, err, content)
	}

	return response.HTML, nil
}
