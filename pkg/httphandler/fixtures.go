package httphandler

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (a Service) fixturesHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := strings.Trim(r.URL.Path, "/")
		if len(query) == 0 {
			httperror.NotFound(w)
			return
		}

		urlParts := strings.Split(query, "/")

		if len(urlParts) == 1 {
			fixturesList, err := a.mailerService.ListFixtures(urlParts[0])
			if httperror.HandleError(w, err) {
				return
			}

			httpjson.WriteArray(w, http.StatusOK, fixturesList)
			return
		}

		if len(urlParts) == 2 {
			content, err := a.mailerService.GetFixture(urlParts[0], urlParts[1])
			if httperror.HandleError(w, err) {
				return
			}

			httpjson.Write(w, http.StatusOK, content)
		}

		httperror.NotFound(w)
	})
}
