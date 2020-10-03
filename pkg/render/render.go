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
	"github.com/ViBiOh/httputils/v3/pkg/templates"
	"github.com/ViBiOh/mailer/pkg/fixtures"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/model"
)

const (
	templatesDir   = "./templates/"
	templateSuffix = ".html"
)

var (
	errTemplateNotFound = errors.New("template not found")
)

// App of package
type App interface {
	Handler() http.Handler
}

type app struct {
	tpl *template.Template

	mjmlApp   mjml.App
	senderApp model.Sender
}

// New creates new App
func New(mjmlApp mjml.App, senderApp model.Sender) App {
	appTemplates, err := templates.GetTemplates(templatesDir, templateSuffix)
	if err != nil {
		logger.Error("%s", err)
	}

	return &app{
		mjmlApp:   mjmlApp,
		senderApp: senderApp,
		tpl: template.Must(template.New("mailer").Funcs(template.FuncMap{
			"odd": func(i int) bool {
				return i%2 == 0
			},
			"split": func(s string) []string {
				return strings.Split(s, "\n")
			},
		}).ParseFiles(appTemplates...)),
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
