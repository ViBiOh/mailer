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

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

func (a App) renderHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if query.IsRoot(r) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			httpjson.WriteArray(w, http.StatusOK, a.mailerApp.ListTemplates())
			return
		}

		if !(r.Method == http.MethodGet || r.Method == http.MethodPost) {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		mr := parseMailRequest(r)

		content, err := a.getContent(r, mr.Tpl)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		mr = mr.Data(content)
		output, err := a.mailerApp.Render(r.Context(), mr)
		if httperror.HandleError(w, err) {
			return
		}

		if r.Method == http.MethodGet {
			writeOutput(w, output)
			return
		}

		a.sendOutput(r.Context(), w, mr, output)
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

func (a App) sendOutput(ctx context.Context, w http.ResponseWriter, mr model.MailRequest, output io.Reader) {
	if err := mr.Check(); err != nil {
		httperror.HandleError(w, httpModel.WrapInvalid(err))
		return
	}

	if httperror.HandleError(w, a.mailerApp.Send(ctx, mr.ConvertToMail(output))) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseMailRequest(r *http.Request) model.MailRequest {
	mr := model.NewMailRequest()

	mr = mr.Template(strings.Trim(r.URL.Path, "/"))
	mr = mr.From(strings.TrimSpace(r.URL.Query().Get("from")))
	mr = mr.As(strings.TrimSpace(r.URL.Query().Get("sender")))
	mr = mr.WithSubject(strings.TrimSpace(r.URL.Query().Get("subject")))

	for _, rawTo := range r.URL.Query()["to"] {
		if cleanTo := strings.TrimSpace(rawTo); len(cleanTo) != 0 {
			mr = mr.To(cleanTo)
		}
	}

	return mr
}
