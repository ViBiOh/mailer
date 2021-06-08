package smtp

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"sync"

	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/mailer/pkg/model"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(nil)
		},
	}
)

// App of package
type App interface {
	Send(ctx context.Context, mail model.Mail) error
}

// Config of package
type Config struct {
	address  *string
	username *string
	password *string
	host     *string
}

type app struct {
	auth    smtp.Auth
	address string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		address:  flags.New(prefix, "smtp").Name("Address").Default("localhost:25").Label("Address").ToString(fs),
		username: flags.New(prefix, "smtp").Name("Username").Default("").Label("Plain Auth Username").ToString(fs),
		password: flags.New(prefix, "smtp").Name("Password").Default("").Label("Plain Auth Password").ToString(fs),
		host:     flags.New(prefix, "smtp").Name("Host").Default("localhost").Label("Plain Auth host").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	var auth smtp.Auth

	user := strings.TrimSpace(*config.username)
	if len(user) > 0 {
		auth = smtp.PlainAuth("", user, strings.TrimSpace(*config.password), strings.TrimSpace(*config.host))
	}

	return &app{
		address: strings.TrimSpace(*config.address),
		auth:    auth,
	}
}

func (a app) Send(_ context.Context, mail model.Mail) error {
	body := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(body)
	body.Reset()

	content, err := io.ReadAll(mail.Content)
	if err != nil {
		return fmt.Errorf("unable to read content: %s", err)
	}

	body.WriteString(fmt.Sprintf("From: %s <%s>\r\n", mail.Sender, mail.From))
	body.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(mail.To, ",")))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", mail.Subject))
	body.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	body.WriteString(fmt.Sprintf("\r\n%s\r\n", content))

	return smtp.SendMail(a.address, a.auth, mail.From, mail.To, body.Bytes())
}
