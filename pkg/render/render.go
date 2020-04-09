package render

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/query"
	"github.com/ViBiOh/httputils/v3/pkg/request"
	"github.com/ViBiOh/httputils/v3/pkg/swagger"
	"github.com/ViBiOh/httputils/v3/pkg/templates"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mailjet"
	"github.com/ViBiOh/mailer/pkg/mjml"
)

const (
	templatesDir   = "./templates/"
	templateSuffix = ".html"
)

var (
	_ swagger.Provider = app{}.Swagger
)

// App of package
type App interface {
	Handler() http.Handler
	Swagger() (swagger.Configuration, error)
}

type app struct {
	mjmlApp    *mjml.App
	mailjetApp *mailjet.App
	tpl        *template.Template
}

// New creates new App
func New(mjmlApp *mjml.App, mailjetApp *mailjet.App) App {
	templates, err := templates.GetTemplates(templatesDir, templateSuffix)
	if err != nil {
		logger.Error("%s", err)
	}

	return &app{
		mjmlApp:    mjmlApp,
		mailjetApp: mailjetApp,
		tpl: template.Must(template.New("mailer").Funcs(template.FuncMap{
			"odd": func(i int) bool {
				return i%2 == 0
			},
		}).ParseFiles(templates...)),
	}
}

func (a app) getBodyContent(r *http.Request) (map[string]interface{}, error) {
	rawContent, err := request.ReadBodyRequest(r)
	if err != nil {
		return nil, err
	}

	if query.GetBool(r, "dump") {
		logger.Info("Payload for %s: %s", r.URL.Path, rawContent)
	}

	var content map[string]interface{}
	if err := json.Unmarshal(rawContent, &content); err != nil {
		return nil, err
	}

	return content, nil
}

func (a app) getContent(templateName string, r *http.Request) (map[string]interface{}, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get("fixture")
		if fixtureName == "" {
			fixtureName = "default"
		}

		return fixtures.Get(templateName, fixtureName)
	}

	return a.getBodyContent(r)
}

func (a app) handleMjml(ctx context.Context, content *bytes.Buffer) error {
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
		return err
	}

	return nil
}

func (a app) listTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	templatesList := make([]string, 0)

	for _, tpl := range a.tpl.Templates() {
		if strings.HasSuffix(tpl.Name(), templateSuffix) {
			templatesList = append(templatesList, strings.TrimSuffix(tpl.Name(), templateSuffix))
		}
	}

	httpjson.ResponseArrayJSON(w, http.StatusOK, templatesList, httpjson.IsPretty(r))
}

// Handler for Render request. Should be use with net/http
func (a app) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ctx := r.Context()

		if r.URL.Path == "" || r.URL.Path == "/" {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				a.listTemplatesHandler(w, r)
			}

			return
		}

		var mail mailjet.Mail
		if r.Method == http.MethodPost {
			mail = a.mailjetApp.GetParameters(r)
			if err := a.mailjetApp.CheckParameters(mail); err != nil {
				httperror.BadRequest(w, err)
				return
			}
		}

		templateName := strings.Trim(r.URL.Path, "/")
		tpl := a.tpl.Lookup(fmt.Sprintf("%s%s", templateName, templateSuffix))

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

		output := CreateWriter()

		if err := templates.ResponseHTMLTemplate(tpl, output, content, http.StatusOK); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if err := a.handleMjml(ctx, output.Content()); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if r.Method == http.MethodGet {
			if _, err := output.WriteResponse(w); err != nil {
				httperror.InternalServerError(w, err)
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

// Swagger exposes swagger configuration for API
func (a app) Swagger() (swagger.Configuration, error) {
	return swagger.Configuration{
		Paths: `/render:
  get:
    description: List templates

    responses:
      200:
        description: List of availables templates
        content:
          application/json:
            schema:
              type: object
              properties:
                results:
                  type: array
                  description: Templates' name
                  items:
                    type: string

/render/{template}:
  parameters:
    - name: template
      in: path
      description: Template's name
      required: true
      schema:
        type: string

  get:
    description: Render template
    parameters:
      - name: fixture
        in: query
        description: Fixture's name
        schema:
          type: string

    responses:
      200:
        description: Render template for fixture
        content:
          text/html:
            type: string

  post:
    description: Render and send template
    parameters:
      - name: from
        in: query
        description: From value of email
        schema:
          type: string
      - name: sender
        in: query
        description: Sender name of email
        schema:
          type: string
      - name: subject
        in: query
        description: Email subject
        schema:
          type: string
      - name: to
        in: query
        description: Recipients of email, comma separated
        schema:
          type: string
    requestBody:
      description: Payload of content
      required: true
      content:
        application/json:
          schema:
            type: object

    responses:
      200:
        description: Render and send template
      400:
        description: Invalid parameters for sending
      500:
        $ref: '#/components/schemas/Error'`,
	}, nil
}
