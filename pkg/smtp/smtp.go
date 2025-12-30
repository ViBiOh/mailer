package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
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

type Service struct {
	auth    smtp.Auth
	tracer  trace.Tracer
	address string
	host    string
}

type Config struct {
	Address  string
	Username string
	Password string
	Host     string
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("Address", "Address").Prefix(prefix).DocPrefix("smtp").StringVar(fs, &config.Address, "127.0.0.1:25", nil)
	flags.New("Username", "Plain Auth Username").Prefix(prefix).DocPrefix("smtp").StringVar(fs, &config.Username, "", nil)
	flags.New("Password", "Plain Auth Password").Prefix(prefix).DocPrefix("smtp").StringVar(fs, &config.Password, "", nil)
	flags.New("Host", "Plain Auth host").Prefix(prefix).DocPrefix("smtp").StringVar(fs, &config.Host, "127.0.0.1", nil)

	return &config
}

func New(config *Config, meterProvider metric.MeterProvider, tracerProvider trace.TracerProvider) Service {
	var auth smtp.Auth

	if len(config.Username) > 0 {
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	}

	mailer_metric.Create(meterProvider, "mailer.smtp")

	service := Service{
		address: config.Address,
		auth:    auth,
		host:    config.Host,
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("smtp")
	}

	return service
}

func (s Service) Send(ctx context.Context, mail model.Mail) error {
	var err error

	_, end := telemetry.StartSpan(ctx, s.tracer, "send")
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

	err = SendMail(s.address, s.host, s.auth, mail.From, mail.To, body.Bytes())

	if err != nil {
		mailer_metric.Increase(ctx, "smtp", "error")
	} else {
		mailer_metric.Increase(ctx, "smtp", "success")
	}

	return err
}

func SendMail(addr, host string, auth smtp.Auth, from string, to []string, body []byte) (err error) {
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

	defer func() {
		if smtpErr := smtpClient.Close(); smtpErr != nil {
			if err != nil {
				err = errors.Join(err, smtpErr)
			}

			err = smtpErr
		}
	}()

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
