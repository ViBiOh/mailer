package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/fibr/pkg/utils"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/httpjson"
	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/templates"
	"github.com/ViBiOh/httputils/pkg/writer"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
)

const (
	templatesDir   = `./templates/`
	templateSuffix = `.html`
)

// App stores informations
type App struct {
	mjmlApp    *mjml.App
	mailjetApp *mailjet.App
	tpl        *template.Template
}

func listFilesByExt(dir, ext string) ([]string, error) {
	output := make([]string, 0)

	if err := filepath.Walk(dir, func(walkedPath string, info os.FileInfo, _ error) error {
		if path.Ext(info.Name()) == ext {
			output = append(output, walkedPath)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf(`Error while listing files: %v`, err)
	}

	return output, nil
}

// NewApp creates new App from Flags' config
func NewApp(mjmlApp *mjml.App, mailjetApp *mailjet.App) *App {
	templates, err := utils.ListFilesByExt(templatesDir, templateSuffix)
	if err != nil {
		log.Fatalf(`Error while getting templates: %v`, err)
	}

	return &App{
		mjmlApp:    mjmlApp,
		mailjetApp: mailjetApp,
		tpl: template.Must(template.New(`mailer`).Funcs(template.FuncMap{
			`odd`: func(i int) bool {
				return i%2 == 0
			},
		}).ParseFiles(templates...)),
	}
}

func (a App) getBodyContent(r *http.Request) (map[string]interface{}, error) {
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

func (a App) getContent(templateName string, r *http.Request) (map[string]interface{}, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get(`fixture`)
		if fixtureName == `` {
			fixtureName = `default`
		}

		return fixtures.Get(templateName, fixtureName)
	}

	return a.getBodyContent(r)
}

func (a App) handleMjml(ctx context.Context, content *bytes.Buffer) error {
	if a.mjmlApp == nil {
		return nil
	}

	payload := content.Bytes()
	if !mjml.IsMJML(payload) {
		return nil
	}

	output, err := a.mjmlApp.Render(ctx, string(payload))
	if err != nil {
		return fmt.Errorf(`Error while converting MJML template: %v`, err)
	}

	content.Reset()
	if _, err := content.WriteString(output); err != nil {
		return fmt.Errorf(`Error while replacing MJML content: %v`, err)
	}

	return nil
}

func (a App) listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templatesList := make([]string, 0)

	for _, tpl := range a.tpl.Templates() {
		if strings.HasSuffix(tpl.Name(), templateSuffix) {
			templatesList = append(templatesList, strings.TrimSuffix(tpl.Name(), templateSuffix))
		}
	}

	if err := httpjson.ResponseArrayJSON(w, http.StatusOK, templatesList, httpjson.IsPretty(r)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

// Handler for Render request. Should be use with net/http
func (a App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		if r.URL.Path == `` || r.URL.Path == `/` {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				a.listTemplatesHandler(w, r)
			}

			return
		}

		var mail *mailjet.Mail
		if r.Method == http.MethodPost {
			mail = a.mailjetApp.GetParameters(r)
			if err := a.mailjetApp.CheckParameters(mail); err != nil {
				httperror.BadRequest(w, err)
				return
			}
		}

		templateName := strings.Trim(r.URL.Path, `/`)
		tpl := a.tpl.Lookup(fmt.Sprintf(`%s%s`, templateName, templateSuffix))

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

		if err := a.handleMjml(ctx, output.Content()); err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while handling MJML: %v`, err))
			return
		}

		if r.Method == http.MethodGet {
			if _, err := output.WriteResponse(w); err != nil {
				httperror.InternalServerError(w, fmt.Errorf(`Error while writing output to response: %v`, err))
			}
			return
		}

		if err := a.mailjetApp.SendMail(ctx, mail, output.Content().String()); err != nil {
			if err == mailjet.ErrEmptyFrom || err == mailjet.ErrEmptyTo || err == mailjet.ErrBlankTo {
				httperror.BadRequest(w, err)
			} else {
				httperror.InternalServerError(w, err)
			}
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
