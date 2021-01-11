package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ViBiOh/mailer/pkg/model"
	"github.com/streadway/amqp"
)

func TestEnabled(t *testing.T) {
	var cases = []struct {
		intention string
		instance  app
		want      bool
	}{
		{
			"empty",
			app{},
			false,
		},
		{
			"simple",
			app{
				url: "http://mailer",
			},
			true,
		},
		{
			"amqp",
			app{
				amqpConnection: &amqp.Connection{},
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.Enabled(); got != tc.want {
				t.Errorf("Enabled() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestSend(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}))
	defer testServer.Close()

	type args struct {
		mailRequest model.MailRequest
	}

	var cases = []struct {
		intention string
		instance  app
		args      args
		wantErr   error
	}{
		{
			"not enabled",
			app{},
			args{
				mailRequest: *model.NewMailRequest(),
			},
			ErrNotEnabled,
		},
		{
			"invalid request",
			app{
				url: "http://mailer",
			},
			args{
				mailRequest: *model.NewMailRequest(),
			},
			errors.New("from email is required"),
		},
		{
			"invalid http",
			app{
				url: testServer.URL,
			},
			args{
				mailRequest: *model.NewMailRequest().From("alice@localhost").To("bob@localhost"),
			},
			errors.New("HTTP/401"),
		},
		{
			"http",
			app{
				url:      testServer.URL,
				name:     "admin",
				password: "password",
			},
			args{
				mailRequest: *model.NewMailRequest().From("alice@localhost").To("bob@localhost"),
			},
			nil,
		},
		{
			"invalid marshal",
			app{
				amqpConnection: &amqp.Connection{},
			},
			args{
				mailRequest: *model.NewMailRequest().From("alice@localhost").To("bob@localhost").Data(func() {}),
			},
			errors.New("unable to marshal mail"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			gotErr := tc.instance.Send(context.Background(), tc.args.mailRequest)

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("Send() = `%s`, want `%s`", gotErr, tc.wantErr)
			}
		})
	}
}
