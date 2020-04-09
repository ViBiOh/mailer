package client

import "context"

// Email describes an email to be sent
type Email struct {
	mailer App

	template   string
	from       string
	sender     string
	subject    string
	recipients []string
	payload    interface{}
}

// NewEmail create a new email
func NewEmail(mailer App) *Email {
	return &Email{
		mailer: mailer,
	}
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
	if e.recipients == nil {
		e.recipients = make([]string, 0)
	}

	e.recipients = append(e.recipients, recipients...)

	return e
}

// Data set payload
func (e *Email) Data(data interface{}) *Email {
	e.payload = data

	return e
}

// Send email
func (e *Email) Send(ctx context.Context) error {
	if e.mailer == nil {
		return nil
	}

	return e.mailer.SendEmail(ctx, *e)
}
