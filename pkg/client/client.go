package client

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/tools"
)

// App stores informations
type App struct {
	url    string
	header http.Header
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) *App {
	user := strings.TrimSpace(*config[`user`])
	pass := strings.TrimSpace(*config[`pass`])

	if user == `` || pass == `` {
		return &App{}
	}

	return &App{
		url: strings.TrimSpace(*config[`url`]),
		header: http.Header{
			`Authorization`: []string{request.GenerateBasicAuth(user, pass)},
		},
	}
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`url`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sURL`, prefix)), `https://mailer.vibioh.fr`, `[mailer] Mailer URL`),
		`user`: flag.String(tools.ToCamel(fmt.Sprintf(`%sUser`, prefix)), ``, `[mailer] Mailer User`),
		`pass`: flag.String(tools.ToCamel(fmt.Sprintf(`%sPass`, prefix)), ``, `[mailer] Mailer Pass`),
	}
}

func (a App) isEnabled() bool {
	return a.url != ``
}

// SendEmail sends emails with Mailer for defined parameters
func (a App) SendEmail(ctx context.Context, template, from, sender, subject string, recipients []string, payload interface{}) error {
	if !a.isEnabled() {
		return nil
	}

	if len(recipients) == 0 {
		return errors.New(`recipients are required`)
	}

	strRecipients := strings.Join(recipients, `,`)
	if strRecipients == `` {
		return errors.New(`no recipient found`)
	}

	_, err := request.DoJSON(ctx, fmt.Sprintf(`%s/render/%s?from=%s&sender=%s&to=%s&subject=%s`, a.url, url.QueryEscape(template), url.QueryEscape(from), url.QueryEscape(sender), url.QueryEscape(strRecipients), url.QueryEscape(subject)), payload, a.header, http.MethodPost)
	if err != nil {
		return err
	}

	return nil
}
