package httphandler

import (
	"io"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/httpjson"
	httpModel "github.com/ViBiOh/httputils/v3/pkg/model"
	"github.com/ViBiOh/httputils/v3/pkg/query"
	"github.com/ViBiOh/mailer/pkg/model"
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

		mailRequest := parseMailRequest(r)

		content, err := a.getContent(r, mailRequest.Tpl)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		mailRequest.Data(content)
		output, err := a.mailerApp.Render(r.Context(), *mailRequest)
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

		if err := mailRequest.Check(); err != nil {
			httperror.HandleError(w, httpModel.WrapInvalid(err))
			return
		}

		if httperror.HandleError(w, a.mailerApp.Send(r.Context(), mailRequest.ConvertToMail(output))) {
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func parseMailRequest(r *http.Request) *model.MailRequest {
	mailRequest := model.NewMailRequest()

	mailRequest.Template(strings.Trim(r.URL.Path, "/"))
	mailRequest.From(strings.TrimSpace(r.URL.Query().Get("from")))
	mailRequest.As(strings.TrimSpace(r.URL.Query().Get("sender")))
	mailRequest.WithSubject(strings.TrimSpace(r.URL.Query().Get("subject")))

	for _, rawTo := range strings.Split(r.URL.Query().Get("to"), ",") {
		if cleanTo := strings.TrimSpace(rawTo); cleanTo != "" {
			mailRequest.To(cleanTo)
		}
	}

	return mailRequest
}
