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
	"github.com/ViBiOh/mailer/pkg/metric"
	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(nil)
		},
	}
)

// App of package
type App struct {
	auth    smtp.Auth
	address string
}

// Config of package
type Config struct {
	address  *string
	username *string
	password *string
	host     *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		address:  flags.New(prefix, "smtp", "Address").Default("localhost:25", nil).Label("Address").ToString(fs),
		username: flags.New(prefix, "smtp", "Username").Default("", nil).Label("Plain Auth Username").ToString(fs),
		password: flags.New(prefix, "smtp", "Password").Default("", nil).Label("Plain Auth Password").ToString(fs),
		host:     flags.New(prefix, "smtp", "Host").Default("localhost", nil).Label("Plain Auth host").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, prometheusRegisterer prometheus.Registerer) App {
	var auth smtp.Auth

	user := strings.TrimSpace(*config.username)
	if len(user) > 0 {
		auth = smtp.PlainAuth("", user, *config.password, strings.TrimSpace(*config.host))
	}

	metric.Create(prometheusRegisterer, "smtp")

	return App{
		address: strings.TrimSpace(*config.address),
		auth:    auth,
	}
}

// Send email by smtp
func (a App) Send(_ context.Context, mail model.Mail) error {
	body := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(body)
	body.Reset()

	fmt.Fprintf(body, "From: %s <%s>\r\n", mail.Sender, mail.From)
	fmt.Fprintf(body, "To: %s\r\n", strings.Join(mail.To, ","))
	fmt.Fprintf(body, "Subject: %s\r\n", mail.Subject)
	body.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	body.WriteString("\r\n")

	if _, err := io.Copy(body, mail.Content); err != nil {
		return fmt.Errorf("unable to read mail content: %s", err)
	}

	body.WriteString("\r\n")

	err := smtp.SendMail(a.address, a.auth, mail.From, mail.To, body.Bytes())

	if err != nil {
		metric.Increase("smtp", "error")
	} else {
		metric.Increase("smtp", "success")
	}

	return err
}
