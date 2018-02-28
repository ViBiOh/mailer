package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/httperror"
	"github.com/ViBiOh/httputils/templates"
	"github.com/ViBiOh/httputils/writer"
	"github.com/ViBiOh/mailer/mjml"
)

var mjmlPrefix = []byte(`<mjml>`)

// App stores informations
type App struct {
	mjmlApp *mjml.App
	tpl     *template.Template
}

// NewApp creates new App from Flags' config
func NewApp(mjmlAppDep *mjml.App) *App {
	return &App{
		mjmlApp: mjmlAppDep,
		tpl:     template.Must(template.New(`mailer`).ParseGlob(`./templates/*.gohtml`)),
	}
}

// Handler for Render request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templateName := strings.Trim(r.URL.Path, `/`)

		tpl := a.tpl.Lookup(fmt.Sprintf(`%s.gohtml`, templateName))

		if tpl == nil {
			httperror.NotFound(w)
			return
		}

		rawContent, err := ioutil.ReadFile(fmt.Sprintf(`./templates/%s/default.json`, templateName))
		if err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while reading default fixture: %v`, err))
			return
		}

		var content map[string]interface{}
		if err := json.Unmarshal(rawContent, &content); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while unmarshalling default fixture: %v`, rawContent, err))
			return
		}

		output := writer.Create()

		if err := templates.WriteHTMLTemplate(tpl, output, content, http.StatusOK); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while writing template: %v`, err))
			return
		}

		renderedTemplate := output.Content().Bytes()
		if bytes.HasPrefix(renderedTemplate, mjmlPrefix) {
			mjmlTemplate, err := a.mjmlApp.Render(string(renderedTemplate))
			if err != nil {
				httperror.BadRequest(w, fmt.Errorf(`Error while converting MJML template: %v`, err))
				return
			}

			output.Content().Reset()
			if _, err := output.Content().WriteString(mjmlTemplate); err != nil {
				httperror.InternalServerError(w, fmt.Errorf(`Error while replacing output content: %v`, err))
				return
			}
		}

		if _, err := output.WriteResponse(w); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while writing output to response: %v`, err))
			return
		}
	})
}
