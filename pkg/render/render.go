package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/utils"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/httpjson"
	"github.com/ViBiOh/httputils/pkg/logger"
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

// App of package
type App struct {
	mjmlApp    *mjml.App
	mailjetApp *mailjet.App
	tpl        *template.Template
}

// New creates new App
func New(mjmlApp *mjml.App, mailjetApp *mailjet.App) *App {
	templates, err := utils.ListFilesByExt(templatesDir, templateSuffix)
	if err != nil {
		logger.Error(`%+v`, err)
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
	rawContent, err := request.ReadBodyRequest(r)
	if err != nil {
		return nil, err
	}

	var content map[string]interface{}
	if err := json.Unmarshal(rawContent, &content); err != nil {
		return nil, errors.WithStack(err)
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
		return err
	}

	content.Reset()
	if _, err := content.WriteString(output); err != nil {
		return errors.WithStack(err)
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
				httperror.InternalServerError(w, err)
			}
			return
		}

		output := writer.Create()

		if err := templates.WriteHTMLTemplate(tpl, output, content, http.StatusOK); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if err := a.handleMjml(ctx, output.Content()); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if r.Method == http.MethodGet {
			if _, err := output.WriteResponse(w); err != nil {
				httperror.InternalServerError(w, errors.WithStack(err))
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
