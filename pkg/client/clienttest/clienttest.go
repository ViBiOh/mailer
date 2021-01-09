package clienttest

import (
	"context"
	"errors"
	"reflect"

	"github.com/ViBiOh/mailer/pkg/client"
	"github.com/ViBiOh/mailer/pkg/model"
)

var (
	_ client.App = App{}
)

// App mock for client
type App struct {
	enabled bool
}

// New creates new App from Config
func New(enabled bool) App {
	return App{
		enabled: enabled,
	}
}

// Enabled mocked implementation
func (a App) Enabled() bool {
	return a.enabled
}

// Send mocked implementation
func (a App) Send(ctx context.Context, email model.MailRequest) error {
	if !a.Enabled() {
		return nil
	}

	if ctx == context.TODO() {
		return errors.New("invalid context")
	}

	if reflect.DeepEqual(email, model.EmptyMailRequest) {
		return errors.New("email is required")
	}

	return nil
}

// Close mocked ressources
func (a App) Close() {
}
