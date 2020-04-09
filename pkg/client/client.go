package client

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/request"
)

// App of package
type App interface {
	Enabled() bool
	SendEmail(context.Context, Email) error
}

// Config of package
type Config struct {
	url  *string
	user *string
	pass *string
}

type app struct {
	url  string
	user string
	pass string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		url:  flags.New(prefix, "mailer").Name("URL").Default("").Label("URL (an instance of github.com/ViBiOh/mailer)").ToString(fs),
		user: flags.New(prefix, "mailer").Name("User").Default("").Label("User").ToString(fs),
		pass: flags.New(prefix, "mailer").Name("Pass").Default("").Label("Pass").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	url := strings.TrimSpace(*config.url)

	if url == "" {
		return &app{}
	}

	return &app{
		url: strings.TrimSpace(*config.url),

		user: strings.TrimSpace(*config.user),
		pass: strings.TrimSpace(*config.pass),
	}
}

func (a app) Enabled() bool {
	return a.url != ""
}

// SendEmail sends emails with Mailer for defined parameters
func (a app) SendEmail(ctx context.Context, email Email) error {
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

	url := fmt.Sprintf("%s/render/%s?from=%s&sender=%s&to=%s&subject=%s", a.url, url.QueryEscape(email.template), url.QueryEscape(email.from), url.QueryEscape(email.sender), url.QueryEscape(strRecipients), url.QueryEscape(email.subject))

	req := request.New().Post(url)
	if a.pass != "" {
		req.BasicAuth(a.user, a.pass)
	}

	_, err := req.JSON(ctx, email.payload)
	return err
}
