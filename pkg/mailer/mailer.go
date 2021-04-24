package mailer

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/templates"
	"github.com/ViBiOh/mailer/pkg/mjml"
	"github.com/ViBiOh/mailer/pkg/model"
)

const (
	templateExtension = ".html"
	jsonExtension     = ".json"
)

// App of package
type App interface {
	Render(context.Context, model.MailRequest) (io.Reader, error)
	Send(context.Context, model.Mail) error
	ListTemplates() []string
	ListFixtures(string) ([]string, error)
	GetFixture(string, string) (map[string]interface{}, error)
}

// Config of package
type Config struct {
	templatesDir *string
}

type app struct {
	tpl *template.Template

	mjmlApp   mjml.App
	senderApp model.Sender

	templatesDir string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		templatesDir: flags.New(prefix, "mailer").Name("Templates").Default("./templates/").Label("Templates directory").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, mjmlApp mjml.App, senderApp model.Sender) App {
	templatesDir := strings.TrimSpace(*config.templatesDir)

	logger.WithField("dir", templatesDir).WithField("extension", templateExtension).Info("Loading templates...")
	appTemplates, err := templates.GetTemplates(templatesDir, templateExtension)
	if err != nil {
		logger.Error("%s", err)
	}

	return &app{
		templatesDir: templatesDir,
		tpl: template.Must(template.New("mailer").Funcs(template.FuncMap{
			"odd": func(i int) bool {
				return i%2 == 0
			},
			"split": func(s string) []string {
				return strings.Split(s, "\n")
			},
		}).ParseFiles(appTemplates...)),

		mjmlApp:   mjmlApp,
		senderApp: senderApp,
	}
}

func (a app) Render(ctx context.Context, mailRequest model.MailRequest) (io.Reader, error) {
	tpl := a.tpl.Lookup(fmt.Sprintf("%s%s", mailRequest.Tpl, templateExtension))
	if tpl == nil {
		return nil, httpModel.ErrNotFound
	}

	buffer := bytes.NewBuffer(nil)
	if err := tpl.Execute(buffer, mailRequest.Payload); err != nil {
		return nil, err
	}

	if err := a.convertMjml(ctx, buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (a app) Send(ctx context.Context, mail model.Mail) error {
	return a.senderApp.Send(ctx, mail)
}

func (a app) ListTemplates() []string {
	var templatesList []string

	for _, tpl := range a.tpl.Templates() {
		if strings.HasSuffix(tpl.Name(), templateExtension) {
			templatesList = append(templatesList, strings.TrimSuffix(tpl.Name(), templateExtension))
		}
	}

	return templatesList
}

func (a app) convertMjml(ctx context.Context, content *bytes.Buffer) error {
	if a.mjmlApp == nil {
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
