package httphandler

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
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
			if httperror.HandleError(w, err) {
				return
			}

			httpjson.ResponseArrayJSON(w, http.StatusOK, fixturesList, httpjson.IsPretty(r))
			return
		}

		if len(urlParts) == 2 {
			content, err := a.mailerApp.GetFixture(urlParts[0], urlParts[1])
			if httperror.HandleError(w, err) {
				return
			}

			httpjson.ResponseJSON(w, http.StatusOK, content, httpjson.IsPretty(r))
		}

		httperror.NotFound(w)
	})
}
