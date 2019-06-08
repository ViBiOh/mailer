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

// App of package
type App interface {
	Enabled() bool
	SendEmail(context.Context, *Email) error
}

// Config of package
type Config struct {
	url  *string
	user *string
	pass *string
}

type app struct {
	url    string
	header http.Header
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:  fs.String(tools.ToCamel(fmt.Sprintf("%sURL", prefix)), "https://mailer.vibioh.fr", "[mailer] Mailer URL"),
		user: fs.String(tools.ToCamel(fmt.Sprintf("%sUser", prefix)), "", "[mailer] Mailer User"),
		pass: fs.String(tools.ToCamel(fmt.Sprintf("%sPass", prefix)), "", "[mailer] Mailer Pass"),
	}
}

// New creates new App from Config
func New(config Config) App {
	user := strings.TrimSpace(*config.user)
	pass := strings.TrimSpace(*config.pass)

	if user == "" || pass == "" {
		return &app{}
	}

	return &app{
		url: strings.TrimSpace(*config.url),
		header: http.Header{
			"Authorization": []string{request.GenerateBasicAuth(user, pass)},
		},
	}
}

func (a app) Enabled() bool {
	return a.url != ""
}

// SendEmail sends emails with Mailer for defined parameters
func (a app) SendEmail(ctx context.Context, email *Email) error {
	if !a.Enabled() {
		return nil
	}

	if len(email.recipients) == 0 {
		return errors.New("recipients are required")
	}

	strRecipients := strings.Join(email.recipients, ",")
	if strRecipients == "" {
		return errors.New("no recipient found")
	}

	_, _, _, err := request.DoJSON(ctx, fmt.Sprintf("%s/render/%s?from=%s&sender=%s&to=%s&subject=%s", a.url, url.QueryEscape(email.template), url.QueryEscape(email.from), url.QueryEscape(email.sender), url.QueryEscape(strRecipients), url.QueryEscape(email.subject)), email.payload, a.header, http.MethodPost)
	if err != nil {
		return err
	}

	return nil
}
