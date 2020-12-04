package client

import (
	"context"
	"errors"
)

var (
	// EmptyEmail for not found case
	EmptyEmail = Email{}
)

// Email describes an email to be sent
type Email struct {
	payload    interface{}
	template   string
	from       string
	sender     string
	subject    string
	recipients []string
}

// NewEmail create a new email
func NewEmail() *Email {
	return &Email{}
}

// Template set template
func (e *Email) Template(template string) *Email {
	e.template = template

	return e
}

// From set from
func (e *Email) From(from string) *Email {
	e.from = from

	return e
}

// As set sender
func (e *Email) As(sender string) *Email {
	e.sender = sender

	return e
}

// WithSubject set subject
func (e *Email) WithSubject(subject string) *Email {
	e.subject = subject

	return e
}

// To add recipients to list
func (e *Email) To(recipients ...string) *Email {
	if len(e.recipients) == 0 {
		e.recipients = recipients
	} else {
		e.recipients = append(e.recipients, recipients...)
	}

	return e
}

// Data set payload
func (e *Email) Data(payload interface{}) *Email {
	e.payload = payload

	return e
}

// Send email
func (e *Email) Send(ctx context.Context, mailer App) error {
	if mailer == nil {
		return errors.New("mailer not provided")
	}

	return mailer.Send(ctx, *e)
}
