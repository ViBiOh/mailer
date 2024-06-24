package httphandler

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
)

func (s Service) HandleFixture(w http.ResponseWriter, r *http.Request) {
	query := strings.Trim(r.PathValue("fixture"), "/")
	if len(query) == 0 {
		httperror.NotFound(r.Context(), w)
		return
	}

	urlParts := strings.Split(query, "/")

	if len(urlParts) == 1 {
		fixturesList, err := s.mailerService.ListFixtures(urlParts[0])
		if httperror.HandleError(r.Context(), w, err) {
			return
		}

		httpjson.WriteArray(r.Context(), w, http.StatusOK, fixturesList)
		return
	}

	if len(urlParts) == 2 {
		content, err := s.mailerService.GetFixture(urlParts[0], urlParts[1])
		if httperror.HandleError(r.Context(), w, err) {
			return
		}

		httpjson.Write(r.Context(), w, http.StatusOK, content)
	}

	httperror.NotFound(r.Context(), w)
}
