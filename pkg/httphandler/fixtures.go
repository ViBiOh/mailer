package httphandler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
	rendererModel "github.com/ViBiOh/httputils/v3/pkg/renderer/model"
)

func (a app) fixturesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := strings.Trim(r.URL.Path, "/")
		if len(query) == 0 {
			httperror.NotFound(w)
			return
		}

		urlParts := strings.Split(query, "/")

		if len(urlParts) == 1 {
			fixturesList, err := a.mailerApp.ListFixtures(urlParts[0])
			if err != nil {
				if errors.Is(err, rendererModel.ErrNotFound) {
					httperror.NotFound(w)
				} else {
					httperror.InternalServerError(w, err)
				}

				return
			}

			httpjson.ResponseArrayJSON(w, http.StatusOK, fixturesList, httpjson.IsPretty(r))
			return
		}

		if len(urlParts) == 2 {
			content, err := a.mailerApp.GetFixture(urlParts[0], urlParts[1])
			if err != nil {
				if errors.Is(err, rendererModel.ErrNotFound) {
					httperror.NotFound(w)
				} else {
					httperror.InternalServerError(w, err)
				}

				return
			}

			httpjson.ResponseJSON(w, http.StatusOK, content, httpjson.IsPretty(r))
		}

		httperror.NotFound(w)
	})
}
