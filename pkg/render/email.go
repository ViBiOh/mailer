package render

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/query"
	"github.com/ViBiOh/httputils/v3/pkg/request"
	"github.com/ViBiOh/httputils/v3/pkg/templates"
	"github.com/ViBiOh/mailer/pkg/fixtures"
)

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

func (a app) getEmailContent(templateName string, r *http.Request) (map[string]interface{}, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get("fixture")
		if fixtureName == "" {
			fixtureName = "default"
		}

		return fixtures.Get(templateName, fixtureName)
	}

	return a.getBodyContent(r)
}

func (a app) getEmailOutput(r *http.Request, templateName string) (*ResponseWriter, error) {
	tpl := a.tpl.Lookup(fmt.Sprintf("%s%s", templateName, templateSuffix))
	if tpl == nil {
		return nil, errTemplateNotFound
	}

	content, err := a.getEmailContent(templateName, r)
	if err != nil {
		return nil, err
	}

	output := CreateWriter()
	if err := templates.ResponseHTMLTemplate(tpl, output, content, http.StatusOK); err != nil {
		return nil, err
	}

	if err := a.convertMjml(r.Context(), output.Content()); err != nil {
		return nil, err
	}

	return output, nil
}

func (a app) sendEmail(w http.ResponseWriter, r *http.Request, content []byte) {
	email := parseEmail(r)
	if err := checkEmail(email); err != nil {
		httperror.BadRequest(w, err)
		return
	}

	if err := a.senderApp.Send(r.Context(), email, content); err != nil {
		httperror.InternalServerError(w, err)
	}
	w.WriteHeader(http.StatusOK)
}
