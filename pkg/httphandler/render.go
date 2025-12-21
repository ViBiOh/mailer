package httphandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"github.com/ViBiOh/mailer/pkg/model"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

func (s Service) HandleRoot(w http.ResponseWriter, r *http.Request) {
	var err error

	_, end := telemetry.StartSpan(r.Context(), s.tracer, "root")
	defer end(&err)

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	httpjson.WriteArray(r.Context(), w, http.StatusOK, s.mailerService.ListTemplates())
}

func (s Service) HandlerTemplate(w http.ResponseWriter, r *http.Request) {
	var err error

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "render")
	defer end(&err)

	mr := parseMailRequest(r)

	fixtureName := r.URL.Query().Get("fixture")
	if fixtureName == "" {
		fixtureName = "default"
	}

	content, err := s.mailerService.GetFixture(fixtureName, fixtureName)
	if err != nil {
		httperror.InternalServerError(r.Context(), w, fmt.Errorf("get content for template `%s`: %w", mr.Tpl, err))
		return
	}

	mr = mr.Data(content)
	output, err := s.mailerService.Render(ctx, mr)
	if httperror.HandleError(ctx, w, err) {
		return
	}

	writeOutput(r.Context(), w, output)
}

func (s Service) HandlerSend(w http.ResponseWriter, r *http.Request) {
	var err error

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "render")
	defer end(&err)

	mr := parseMailRequest(r)

	content, err := httpjson.Parse[map[string]any](r)
	if err != nil {
		httperror.BadRequest(ctx, w, fmt.Errorf("parse content: %w", err))
		return
	}

	mr = mr.Data(content)
	output, err := s.mailerService.Render(ctx, mr)
	if httperror.HandleError(r.Context(), w, err) {
		return
	}

	s.sendOutput(ctx, w, mr, output)
}

func writeOutput(ctx context.Context, w http.ResponseWriter, output io.Reader) {
	w.Header().Add("Content-Type", "text/html; charset=UTF-8")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("X-UA-Compatible", "ie=edge")
	w.WriteHeader(http.StatusOK)

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	if _, err := io.CopyBuffer(w, output, buffer.Bytes()); err != nil {
		httperror.InternalServerError(ctx, w, err)
	}
}

func (s Service) sendOutput(ctx context.Context, w http.ResponseWriter, mr model.MailRequest, output io.Reader) {
	if err := mr.Check(); err != nil {
		httperror.HandleError(ctx, w, httpModel.WrapInvalid(err))
		return
	}

	if httperror.HandleError(ctx, w, s.mailerService.Send(ctx, mr.ConvertToMail(ctx, output))) {
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseMailRequest(r *http.Request) model.MailRequest {
	mr := model.NewMailRequest()

	mr = mr.Template(strings.Trim(r.PathValue("template"), "/"))
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
