package mjml

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/request"
	"github.com/ViBiOh/httputils/tools"
)

const (
	renderURL = `https://api.mjml.io/v1/render`
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
	headers map[string]string
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) *App {
	if *config[`applicationID`] == `` {
		return &App{}
	}

	return &App{
		headers: map[string]string{`Authorization`: request.GetBasicAuth(*config[`applicationID`], *config[`secretKey`])},
	}
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`applicationID`: flag.String(tools.ToCamel(fmt.Sprintf(`%sApplicationID`, prefix)), ``, `[mjml] Application ID`),
		`secretKey`:     flag.String(tools.ToCamel(fmt.Sprintf(`%sSecretKey`, prefix)), ``, `[mjml] Secret Key`),
	}
}

// Render MJML template
func (a *App) Render(template string) (string, error) {
	content, err := request.DoJSON(renderURL, mjmlRequest{template}, a.headers, http.MethodPost)
	if err != nil {
		return ``, fmt.Errorf(`Error while sending data: %v %s`, err, content)
	}

	var response mjmlResponse
	if err := json.Unmarshal(content, &response); err != nil {
		return ``, fmt.Errorf(`Error while unmarshalling data %s: %v`, content, err)
	}

	return response.HTML, nil
}
