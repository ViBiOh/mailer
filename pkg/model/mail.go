package model

import "context"

// Sender send given email
type Sender interface {
	Send(ctx context.Context, mail Mail, html []byte) error
}

// Mail describe envelope of an email
type Mail struct {
	From    string
	Sender  string
	Subject string
	To      []string
}
