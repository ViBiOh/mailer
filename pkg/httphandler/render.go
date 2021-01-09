package httphandler

import (
	"io"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
	"github.com/ViBiOh/httputils/v3/pkg/query"
)

func (a app) renderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if query.IsRoot(r) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			httpjson.ResponseArrayJSON(w, http.StatusOK, a.mailerApp.ListTemplates(), httpjson.IsPretty(r))
			return
		}

		if !(r.Method == http.MethodGet || r.Method == http.MethodPost) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		name := strings.Trim(r.URL.Path, "/")
		content, err := a.getContent(r, name)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		output, err := a.mailerApp.Render(r.Context(), name, content)
		if httperror.HandleError(w, err) {
			return
		}

		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=UTF-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("X-UA-Compatible", "ie=edge")
			w.WriteHeader(http.StatusOK)

			if _, err = io.Copy(w, output); err != nil {
				httperror.InternalServerError(w, err)
			}

			return
		}

		a.sendEmail(w, r, output)
	})
}
