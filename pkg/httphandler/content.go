package httphandler

import (
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (a app) getContent(r *http.Request, name string) (map[string]interface{}, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get("fixture")
		if fixtureName == "" {
			fixtureName = "default"
		}

		return a.mailerApp.GetFixture(name, fixtureName)
	}

	var content map[string]interface{}
	return content, httpjson.Parse(r, &content, "content")
}
