package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/smtp"
	"strings"
	"sync"

	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	mailer_metric "github.com/ViBiOh/mailer/pkg/metric"
	"github.com/ViBiOh/mailer/pkg/model"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(nil)
	},
}

// App of package
type App struct {
	auth    smtp.Auth
	tracer  trace.Tracer
	address string
	host    string
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
		address:  flags.New("Address", "Address").Prefix(prefix).DocPrefix("smtp").String(fs, "localhost:25", nil),
		username: flags.New("Username", "Plain Auth Username").Prefix(prefix).DocPrefix("smtp").String(fs, "", nil),
		password: flags.New("Password", "Plain Auth Password").Prefix(prefix).DocPrefix("smtp").String(fs, "", nil),
		host:     flags.New("Host", "Plain Auth host").Prefix(prefix).DocPrefix("smtp").String(fs, "localhost", nil),
	}
}

// New creates new App from Config
func New(config Config, meter metric.Meter, tracer trace.Tracer) App {
	var auth smtp.Auth

	user := strings.TrimSpace(*config.username)
	if len(user) > 0 {
		auth = smtp.PlainAuth("", user, *config.password, strings.TrimSpace(*config.host))
	}

	mailer_metric.Create(meter, "smtp")

	return App{
		address: strings.TrimSpace(*config.address),
		auth:    auth,
		host:    *config.host,
		tracer:  tracer,
	}
}

// Send email by smtp
func (a App) Send(ctx context.Context, mail model.Mail) error {
	var err error

	_, end := telemetry.StartSpan(ctx, a.tracer, "send")
	defer end(&err)

	body := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(body)
	body.Reset()

	fmt.Fprintf(body, "From: %s <%s>\r\n", mail.Sender, mail.From)
	fmt.Fprintf(body, "To: %s\r\n", strings.Join(mail.To, ","))
	fmt.Fprintf(body, "Subject: %s\r\n", mail.Subject)
	body.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	body.WriteString("\r\n")

	if _, err = io.Copy(body, mail.Content); err != nil {
		return fmt.Errorf("read mail content: %w", err)
	}

	body.WriteString("\r\n")

	err = SendMail(a.address, a.host, a.auth, mail.From, mail.To, body.Bytes())

	if err != nil {
		mailer_metric.Increase(ctx, "smtp", "error")
	} else {
		mailer_metric.Increase(ctx, "smtp", "success")
	}

	return err
}

func SendMail(addr, host string, auth smtp.Auth, from string, to []string, body []byte) error {
	smtpConn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	smtpClient, err := smtp.NewClient(smtpConn, host)
	if err != nil {
		return fmt.Errorf("client: %w", err)
	}

	defer smtpClient.Close()

	if auth != nil {
		if err = smtpClient.Auth(auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}

	if err = smtpClient.Mail(from); err != nil {
		return fmt.Errorf("mail: %w", err)
	}

	for _, recipient := range to {
		if err = smtpClient.Rcpt(recipient); err != nil {
			return fmt.Errorf("recipient `%s`: %w", recipient, err)
		}
	}

	writer, err := smtpClient.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}

	if _, err = writer.Write(body); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return smtpClient.Quit()
}
