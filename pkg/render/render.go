package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/httpjson"
	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/templates"
	"github.com/ViBiOh/httputils/pkg/writer"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mjml"
)

const (
	templateSuffix = `.gohtml`
)

// App stores informations
type App struct {
	mjmlApp *mjml.App
	tpl     *template.Template
}

// NewApp creates new App from Flags' config
func NewApp(mjmlAppDep *mjml.App) *App {
	return &App{
		mjmlApp: mjmlAppDep,
		tpl:     template.Must(template.New(`mailer`).ParseGlob(fmt.Sprintf(`./templates/*%s`, templateSuffix))),
	}
}

func (a *App) getBodyContent(r *http.Request) (map[string]interface{}, error) {
	rawContent, err := request.ReadBody(r.Body)
	if err != nil {
		return nil, fmt.Errorf(`Error while reading body's content: %v`, err)
	}

	var content map[string]interface{}
	if err := json.Unmarshal(rawContent, &content); err != nil {
		return nil, fmt.Errorf(`Error while unmarshalling body's content: %v`, err)
	}

	return content, nil
}

func (a *App) getContent(templateName string, r *http.Request) (map[string]interface{}, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get(`fixture`)
		if fixtureName == `` {
			fixtureName = `default`
		}

		return fixtures.Get(templateName, fixtureName)
	}

	return a.getBodyContent(r)
}

func (a *App) handleMjml(content *bytes.Buffer) error {
	payload := content.Bytes()

	if !mjml.IsMJML(payload) {
		return nil
	}

	output, err := a.mjmlApp.Render(string(payload))
	if err != nil {
		return fmt.Errorf(`Error while converting MJML template: %v`, err)
	}

	content.Reset()
	if _, err := content.WriteString(output); err != nil {
		return fmt.Errorf(`Error while replacing MJML content: %v`, err)
	}

	return nil
}

func (a *App) listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templateList := make([]string, len(a.tpl.Templates()))

	for index, tpl := range a.tpl.Templates() {
		templateList[index] = strings.TrimSuffix(tpl.Name(), templateSuffix)
	}

	if err := httpjson.ResponseArrayJSON(w, http.StatusOK, templateList, httpjson.IsPretty(r.URL.RawQuery)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

// Handler for Render request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == `` || r.URL.Path == `/` {
			a.listTemplatesHandler(w, r)
			return
		}

		templateName := strings.Trim(r.URL.Path, `/`)

		tpl := a.tpl.Lookup(fmt.Sprintf(`%s.gohtml`, templateName))

		if tpl == nil {
			httperror.NotFound(w)
			return
		}

		content, err := a.getContent(templateName, r)
		if err != nil {
			if err == fixtures.ErrNoTemplate {
				httperror.NotFound(w)
			} else {
				httperror.InternalServerError(w, fmt.Errorf(`Error while getting content: %v`, err))
			}
			return
		}

		output := writer.Create()

		if err := templates.WriteHTMLTemplate(tpl, output, content, http.StatusOK); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while writing template: %v`, err))
			return
		}

		if err := a.handleMjml(output.Content()); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while handling MJML: %v`, err))
			return
		}

		if _, err := output.WriteResponse(w); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while writing output to response: %v`, err))
			return
		}
	})
}
