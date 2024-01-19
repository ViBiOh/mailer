package model

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"strings"
)

// MailRequest describes an email to be sent
type MailRequest struct {
	Payload    any
	Tpl        string
	FromEmail  string
	Sender     string
	Subject    string
	Recipients []string
}

// NewMailRequest create a new email
func NewMailRequest() MailRequest {
	return MailRequest{}
}

// Template set template
func (mr MailRequest) Template(template string) MailRequest {
	mr.Tpl = template

	return mr
}

// From set from
func (mr MailRequest) From(fromEmail string) MailRequest {
	mr.FromEmail = fromEmail

	return mr
}

// As set sender
func (mr MailRequest) As(sender string) MailRequest {
	mr.Sender = sender

	return mr
}

// WithSubject set subject
func (mr MailRequest) WithSubject(subject string) MailRequest {
	mr.Subject = subject

	return mr
}

// To add recipients to list
func (mr MailRequest) To(recipients ...string) MailRequest {
	if len(mr.Recipients) == 0 {
		mr.Recipients = recipients
	} else {
		mr.Recipients = append(mr.Recipients, recipients...)
	}

	return mr
}

// Data set payload
func (mr MailRequest) Data(payload any) MailRequest {
	mr.Payload = payload

	return mr
}

// Check checks if current instance is valid
func (mr MailRequest) Check() error {
	if len(mr.FromEmail) == 0 {
		return errors.New("from email is required")
	}

	if len(mr.Recipients) == 0 {
		return errors.New("recipients are required")
	}

	for index, recipient := range mr.Recipients {
		if len(recipient) == 0 {
			return fmt.Errorf("recipient at index %d is empty", index)
		}
	}

	if len(mr.Tpl) == 0 {
		return errors.New("template name is required")
	}

	return nil
}

func getSubject(ctx context.Context, subject string, payload any) string {
	if !strings.Contains(subject, "{{") {
		return subject
	}

	tpl, err := template.New("subject").Parse(subject)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelWarn, "cannot parse template subject", slog.String("subject", subject), slog.Any("error", err))
		return subject
	}

	subjectOutput := strings.Builder{}

	if err := tpl.Execute(&subjectOutput, payload); err != nil {
		slog.LogAttrs(ctx, slog.LevelWarn, "cannot execute template subject", slog.String("subject", subject), slog.Any("error", err))
		return subject
	}

	return subjectOutput.String()
}

// ConvertToMail convert mail request to Mail with given content
func (mr MailRequest) ConvertToMail(ctx context.Context, content io.Reader) Mail {
	return Mail{
		From:    mr.FromEmail,
		Sender:  mr.Sender,
		Subject: getSubject(ctx, mr.Subject, mr.Payload),
		Content: content,
		To:      mr.Recipients,
	}
}

// Mail describe envelope of an email
type Mail struct {
	From    string
	Sender  string
	Subject string
	Content io.Reader
	To      []string
}

// LoggedCloser closes a ressources with handling error
func LoggedCloser(closer io.Closer) {
	if err := closer.Close(); err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "close", slog.Any("error", err))
	}
}
