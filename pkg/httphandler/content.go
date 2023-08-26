package httphandler

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) getContent(r *http.Request, name string) (map[string]any, error) {
	if r.Method == http.MethodGet {
		fixtureName := r.URL.Query().Get("fixture")
		if fixtureName == "" {
			fixtureName = "default"
		}

		return s.mailerService.GetFixture(name, fixtureName)
	}

	var content map[string]any
	if err := httpjson.Parse(r, &content); err != nil {
		return nil, fmt.Errorf("parse content: %w", err)
	}

	return content, nil
}
