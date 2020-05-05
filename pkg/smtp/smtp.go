package smtp

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/mailer/pkg/model"
)

// App of package
type App interface {
	Send(ctx context.Context, mail model.Mail, html []byte) error
}

// Config of package
type Config struct {
	addr     *string
	user     *string
	password *string
	host     *string
}

type app struct {
	addr string
	auth smtp.Auth
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		addr:     flags.New(prefix, "smtp").Name("Address").Default("localhost:25").Label("Address").ToString(fs),
		user:     flags.New(prefix, "smtp").Name("AuthUser").Default("").Label("Plain Auth User").ToString(fs),
		password: flags.New(prefix, "smtp").Name("AuthPassword").Default("").Label("Plain Auth Password").ToString(fs),
		host:     flags.New(prefix, "smtp").Name("AuthHost").Default("localhost").Label("Plain Auth host").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	var auth smtp.Auth

	user := strings.TrimSpace(*config.user)
	if len(user) > 0 {
		auth = smtp.PlainAuth("", user, strings.TrimSpace(*config.password), strings.TrimSpace(*config.host))
	}

	return &app{
		addr: strings.TrimSpace(*config.addr),
		auth: auth,
	}
}

func (a app) Send(_ context.Context, mail model.Mail, html []byte) error {
	body := bytes.Buffer{}
	body.WriteString(fmt.Sprintf("From: %s <%s>\r\n", mail.Sender, mail.From))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", mail.Subject))
	body.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	body.WriteString(fmt.Sprintf("\r\n%s\r\n", html))

	return smtp.SendMail(a.addr, a.auth, mail.From, mail.To, body.Bytes())
}
