package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/ViBiOh/mailer/pkg/metric"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
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
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

// App of package
type App struct {
	senderApp    sender
	tpl          *template.Template
	templatesDir string
	tracer       trace.Tracer
	mjmlApp      mjml.App
}

// Config of package
type Config struct {
	templatesDir *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		templatesDir: flags.New(prefix, "mailer", "Templates").Default("./templates/", nil).Label("Templates directory").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, mjmlApp mjml.App, senderApp sender, prometheusRegisterer prometheus.Registerer, tracerApp tracer.App) App {
	templatesDir := strings.TrimSpace(*config.templatesDir)

	logger.WithField("dir", templatesDir).WithField("extension", templateExtension).Info("Loading templates...")
	appTemplates, err := getTemplates(templatesDir, templateExtension)
	if err != nil {
		logger.Error("%s", err)
	}

	metric.Create(prometheusRegisterer, "render")

	return App{
		templatesDir: templatesDir,
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

		mjmlApp:   mjmlApp,
		senderApp: senderApp,
		tracer:    tracerApp.GetTracer("mailer"),
	}
}

// Enabled checks if requirements are met
func (a App) Enabled() bool {
	return a.tpl != nil
}

// AmqpHandler handler amqp message
func (a App) AmqpHandler(message amqp.Delivery) error {
	ctx := context.Background()

	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "amqp")
		defer span.End()
	}

	var mailRequest model.MailRequest
	if err := json.Unmarshal(message.Body, &mailRequest); err != nil {
		return fmt.Errorf("unable to parse payload: %s", err)
	}

	output, err := a.Render(ctx, mailRequest)
	if err != nil {
		return fmt.Errorf("unable to render email: %s", err)
	}

	return a.Send(ctx, mailRequest.ConvertToMail(output))
}

// Render email
func (a App) Render(ctx context.Context, mailRequest model.MailRequest) (io.Reader, error) {
	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "render", trace.WithSpanKind(trace.SpanKindInternal))
		defer span.End()
	}

	tpl := a.tpl.Lookup(fmt.Sprintf("%s%s", mailRequest.Tpl, templateExtension))
	if tpl == nil {
		return nil, httpModel.ErrNotFound
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()

	if err := tpl.Execute(buffer, mailRequest.Payload); err != nil {
		metric.Increase("render", "error")
		return nil, err
	}

	metric.Increase("render", "success")

	if err := a.convertMjml(ctx, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

// Send email
func (a App) Send(ctx context.Context, mail model.Mail) error {
	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "send", trace.WithSpanKind(trace.SpanKindInternal))
		defer span.End()
	}

	return a.senderApp.Send(ctx, mail)
}

// ListTemplates availables to render
func (a App) ListTemplates() []string {
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

func (a App) convertMjml(ctx context.Context, content *bytes.Buffer) error {
	if !a.mjmlApp.Enabled() {
		return nil
	}

	payload := content.Bytes()
	if !mjml.IsMJML(payload) {
		return nil
	}

	output, err := a.mjmlApp.Render(ctx, string(payload))
	if err != nil {
		return err
	}

	content.Reset()
	if _, err := content.WriteString(output); err != nil {
		return err
	}

	return nil
}
