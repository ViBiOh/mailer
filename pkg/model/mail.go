package model

import (
	"context"
	"io"
)

// Sender send given email
type Sender interface {
	Send(ctx context.Context, mail Mail) error
}

// Mail describe envelope of an email
type Mail struct {
	From    string
	Sender  string
	Subject string
	To      []string
	Content io.Reader
}
