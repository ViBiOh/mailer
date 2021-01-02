package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/query"
	"github.com/ViBiOh/httputils/v3/pkg/request"
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

func (a app) getContent(r *http.Request, name string) (map[string]interface{}, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get("fixture")
		if fixtureName == "" {
			fixtureName = "default"
		}

		return a.mailerApp.GetFixture(name, fixtureName)
	}

	return a.getBodyContent(r)
}
