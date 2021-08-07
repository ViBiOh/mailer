package httphandler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/mailer/pkg/model"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 32*1024))
		},
	}
)

func (a App) renderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if query.IsRoot(r) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			httpjson.WriteArray(w, http.StatusOK, a.mailerApp.ListTemplates(), httpjson.IsPretty(r))
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
			writeOutput(w, output)
			return
		}

		a.sendOutput(r.Context(), w, mailRequest, output)
	})
}

func writeOutput(w http.ResponseWriter, output io.Reader) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("X-UA-Compatible", "ie=edge")
	w.WriteHeader(http.StatusOK)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	if _, err := io.CopyBuffer(w, output, buffer.Bytes()); err != nil {
		httperror.InternalServerError(w, err)
	}
}

func (a App) sendOutput(ctx context.Context, w http.ResponseWriter, mailRequest *model.MailRequest, output io.Reader) {
	if err := mailRequest.Check(); err != nil {
		httperror.HandleError(w, httpModel.WrapInvalid(err))
		return
	}

	if httperror.HandleError(w, a.mailerApp.Send(ctx, mailRequest.ConvertToMail(output))) {
		return
	}
	w.WriteHeader(http.StatusOK)
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
