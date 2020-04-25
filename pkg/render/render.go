package render

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/query"
	"github.com/ViBiOh/httputils/v3/pkg/swagger"
	"github.com/ViBiOh/httputils/v3/pkg/templates"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/smtp"
)

const (
	templatesDir   = "./templates/"
	templateSuffix = ".html"
)

var (
	_ swagger.Provider = app{}.Swagger

	errTemplateNotFound = errors.New("template not found")
)

// App of package
type App interface {
	Handler() http.Handler
	Swagger() (swagger.Configuration, error)
}

type app struct {
	tpl *template.Template

	mjmlApp mjml.App
	smtpApp smtp.App
}

// New creates new App
func New(mjmlApp mjml.App, smtpApp smtp.App) App {
	templates, err := templates.GetTemplates(templatesDir, templateSuffix)
	if err != nil {
		logger.Error("%s", err)
	}

	return &app{
		mjmlApp: mjmlApp,
		smtpApp: smtpApp,
		tpl: template.Must(template.New("mailer").Funcs(template.FuncMap{
			"odd": func(i int) bool {
				return i%2 == 0
			},
			"split": func(s string) []string {
				return strings.Split(s, "\n")
			},
		}).ParseFiles(templates...)),
	}
}

func (a app) convertMjml(ctx context.Context, content *bytes.Buffer) error {
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

// Handler for Render request. Should be use with net/http
func (a app) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkRequest(r) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return

		}

		if query.IsRoot(r) && r.Method == http.MethodGet {
			a.listTemplatesHandler(w, r)
			return
		}

		emailContent, err := a.getEmailOutput(r, strings.Trim(r.URL.Path, "/"))
		if errors.Is(err, errTemplateNotFound) || errors.Is(err, fixtures.ErrNoTemplate) {
			httperror.NotFound(w)
			return
		} else if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		if r.Method == http.MethodGet {
			if _, err := emailContent.WriteResponse(w); err != nil {
				httperror.InternalServerError(w, err)
			}
			return
		}

		a.sendEmail(w, r, emailContent.content.Bytes())
	})
}

func checkRequest(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost:
		return !query.IsRoot(r)
	case http.MethodGet:
		return true
	default:
		return false
	}
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
