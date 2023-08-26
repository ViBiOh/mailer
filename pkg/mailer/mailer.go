package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/ViBiOh/flags"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	mailer_metric "github.com/ViBiOh/mailer/pkg/metric"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/model"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type sender interface {
	Send(ctx context.Context, mail model.Mail) error
}

const (
	templateExtension = ".html"
	jsonExtension     = ".json"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(nil)
	},
}

type Service struct {
	senderService sender
	tpl           *template.Template
	templatesDir  string
	tracer        trace.Tracer
	mjmlService   mjml.Service
}

type Config struct {
	TemplatesDir string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	var config Config

	flags.New("Templates", "Templates directory").Prefix(prefix).DocPrefix("mailer").StringVar(fs, &config.TemplatesDir, "./templates/", nil)

	return config
}

func New(config Config, mjmlService mjml.Service, senderService sender, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) Service {
	slog.Info("Loading templates...", "dir", config.TemplatesDir, "extension", templateExtension)
	appTemplates, err := getTemplates(config.TemplatesDir, templateExtension)
	if err != nil {
		slog.Error("get templates", "err", err)
	}

	mailer_metric.Create(meterProvider, "mailer.render")

	service := Service{
		templatesDir: config.TemplatesDir,
		tpl: template.Must(template.New("mailer").Funcs(template.FuncMap{
			"odd": func(i int) bool {
				return i%2 == 0
			},
			"split": func(s, separator string) []string {
				return strings.Split(s, separator)
			},
			"contains": func(s, substr string) bool {
				return strings.Contains(s, substr)
			},
		}).ParseFiles(appTemplates...)),

		mjmlService:   mjmlService,
		senderService: senderService,
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("mailer")
	}

	return service
}

func (a Service) Enabled() bool {
	return a.tpl != nil
}

func (a Service) AmqpHandler(ctx context.Context, message amqp.Delivery) (err error) {
	ctx, end := telemetry.StartSpan(ctx, a.tracer, "amqp")
	defer end(&err)

	var mailRequest model.MailRequest
	if err := json.Unmarshal(message.Body, &mailRequest); err != nil {
		return fmt.Errorf("parse payload: %w", err)
	}

	output, err := a.Render(ctx, mailRequest)
	if err != nil {
		return fmt.Errorf("render email: %w", err)
	}

	return a.Send(ctx, mailRequest.ConvertToMail(output))
}

func (a Service) Render(ctx context.Context, mailRequest model.MailRequest) (io.Reader, error) {
	var err error

	ctx, end := telemetry.StartSpan(ctx, a.tracer, "render")
	defer end(&err)

	tpl := a.tpl.Lookup(fmt.Sprintf("%s%s", mailRequest.Tpl, templateExtension))
	if tpl == nil {
		return nil, httpModel.ErrNotFound
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()

	if err = tpl.Execute(buffer, mailRequest.Payload); err != nil {
		mailer_metric.Increase(ctx, "render", "error")
		return nil, err
	}

	mailer_metric.Increase(ctx, "render", "success")

	if err = a.convertMjml(ctx, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (a Service) Send(ctx context.Context, mail model.Mail) (err error) {
	ctx, end := telemetry.StartSpan(ctx, a.tracer, "send")
	defer end(&err)

	return a.senderService.Send(ctx, mail)
}

func (a Service) ListTemplates() []string {
	var templatesList []string

	for _, tpl := range a.tpl.Templates() {
		if strings.HasSuffix(tpl.Name(), templateExtension) {
			templatesList = append(templatesList, strings.TrimSuffix(tpl.Name(), templateExtension))
		}
	}

	return templatesList
}

func getTemplates(dir, extension string) ([]string, error) {
	var templates []string

	return templates, filepath.Walk(dir, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path.Ext(filename) == extension {
			templates = append(templates, filename)
		}

		return nil
	})
}

func (a Service) convertMjml(ctx context.Context, content *bytes.Buffer) error {
	if !a.mjmlService.Enabled() {
		return nil
	}

	payload := content.Bytes()
	if !mjml.IsMJML(payload) {
		return nil
	}

	output, err := a.mjmlService.Render(ctx, string(payload))
	if err != nil {
		return err
	}

	content.Reset()
	if _, err := content.WriteString(output); err != nil {
		return err
	}

	return nil
}
