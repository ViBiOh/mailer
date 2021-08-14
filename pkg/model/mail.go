package model

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	// EmptyMailRequest for not found case
	EmptyMailRequest = MailRequest{}
)

// MailRequest describes an email to be sent
type MailRequest struct {
	Payload    interface{}
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
func (mr MailRequest) Template(Tpl string) MailRequest {
	mr.Tpl = Tpl

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
func (mr MailRequest) Data(payload interface{}) MailRequest {
	mr.Payload = payload

	return mr
}

// Check checks if current instance is valid
func (mr MailRequest) Check() error {
	if len(strings.TrimSpace(mr.FromEmail)) == 0 {
		return errors.New("from email is required")
	}

	if len(mr.Recipients) == 0 {
		return errors.New("recipients are required")
	}

	for index, recipient := range mr.Recipients {
		if len(strings.TrimSpace(recipient)) == 0 {
			return fmt.Errorf("recipient at index %d is empty", index)
		}
	}

	return nil
}

func getSubject(subject string, payload interface{}) string {
	if !strings.Contains(subject, "{{") {
		return subject
	}

	tpl, err := template.New("subject").Parse(subject)
	if err != nil {
		logger.Warn("subject `%s` is not a template: %s", subject, err)
		return subject
	}

	subjectOutput := strings.Builder{}

	if err := tpl.Execute(&subjectOutput, payload); err != nil {
		logger.Warn("subject `%s` template got an error: %s", subject, err)
		return subject
	}

	return subjectOutput.String()
}

// ConvertToMail convert mail request to Mail with given content
func (mr MailRequest) ConvertToMail(content io.Reader) Mail {

	return Mail{
		From:    mr.FromEmail,
		Sender:  mr.Sender,
		Subject: getSubject(mr.Subject, mr.Payload),
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
		logger.Error("error while closing: %s", err)
	}
}
