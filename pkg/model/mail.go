package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/streadway/amqp"
)

var (
	// EmptyMailRequest for not found case
	EmptyMailRequest = MailRequest{}
)

// Sender send given email
type Sender interface {
	Send(ctx context.Context, mail Mail) error
}

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
func NewMailRequest() *MailRequest {
	return &MailRequest{}
}

// Template set template
func (mr *MailRequest) Template(Tpl string) *MailRequest {
	mr.Tpl = Tpl

	return mr
}

// From set from
func (mr *MailRequest) From(fromEmail string) *MailRequest {
	mr.FromEmail = fromEmail

	return mr
}

// As set sender
func (mr *MailRequest) As(sender string) *MailRequest {
	mr.Sender = sender

	return mr
}

// WithSubject set subject
func (mr *MailRequest) WithSubject(subject string) *MailRequest {
	mr.Subject = subject

	return mr
}

// To add recipients to list
func (mr *MailRequest) To(recipients ...string) *MailRequest {
	if len(mr.Recipients) == 0 {
		mr.Recipients = recipients
	} else {
		mr.Recipients = append(mr.Recipients, recipients...)
	}

	return mr
}

// Data set payload
func (mr *MailRequest) Data(payload interface{}) *MailRequest {
	mr.Payload = payload

	return mr
}

// Check checks if current instance is valid
func (mr *MailRequest) Check() error {
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

// ConvertToMail convert mail request to Mail with given content
func (mr *MailRequest) ConvertToMail(content io.Reader) Mail {
	return Mail{
		From:    mr.FromEmail,
		Sender:  mr.Sender,
		Subject: mr.Subject,
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

// InitAMQP inits AMQP connection, channel and queue
func InitAMQP(uri string) (*amqp.Connection, *amqp.Channel, amqp.Queue, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, nil, amqp.Queue{}, fmt.Errorf("unable to connect to amqp: %s", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return conn, nil, amqp.Queue{}, fmt.Errorf("unable to open communication channel: %s", err)
	}

	queue, err := channel.QueueDeclare("mailer", true, false, false, false, nil)
	if err != nil {
		return conn, channel, amqp.Queue{}, fmt.Errorf("unable to declare queue: %s", err)
	}

	return conn, channel, queue, nil
}

// LoggedCloser closes a ressources with handling error
func LoggedCloser(closer io.Closer) {
	if err := closer.Close(); err != nil {
		logger.Error("error while closing: %s", err)
	}
}
