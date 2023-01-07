package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/mailer/pkg/model"
)

func TestEnabled(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		instance App
		want     bool
	}{
		"empty": {
			App{},
			false,
		},
		"simple": {
			App{
				req: request.Post("http://mailer"),
			},
			true,
		},
	}

	for intention, testCase := range cases {
		intention, testCase := intention, testCase

		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := testCase.instance.Enabled(); got != testCase.want {
				t.Errorf("Enabled() = %t, want %t", got, testCase.want)
			}
		})
	}
}

func TestSend(t *testing.T) {
	t.Parallel()

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

	cases := map[string]struct {
		instance App
		args     args
		wantErr  error
	}{
		"not enabled": {
			App{},
			args{
				mailRequest: model.NewMailRequest(),
			},
			ErrNotEnabled,
		},
		"invalid request": {
			App{
				req: request.Post("http://mailer"),
			},
			args{
				mailRequest: model.NewMailRequest(),
			},
			errors.New("from email is required"),
		},
		"invalid http": {
			App{
				req: request.Post(testServer.URL),
			},
			args{
				mailRequest: model.NewMailRequest().From("alice@localhost").To("bob@localhost").Template("test"),
			},
			errors.New("HTTP/401"),
		},
		"http": {
			App{
				req: request.Post(testServer.URL).BasicAuth("admin", "password"),
			},
			args{
				mailRequest: model.NewMailRequest().From("alice@localhost").To("bob@localhost").Template("test"),
			},
			nil,
		},
	}

	for intention, testCase := range cases {
		intention, testCase := intention, testCase

		t.Run(intention, func(t *testing.T) {
			gotErr := testCase.instance.Send(context.TODO(), testCase.args.mailRequest)

			failed := false

			if testCase.wantErr == nil && gotErr != nil {
				failed = true
			} else if testCase.wantErr != nil && gotErr == nil {
				failed = true
			} else if testCase.wantErr != nil && !strings.Contains(gotErr.Error(), testCase.wantErr.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("Send() = `%s`, want `%s`", gotErr, testCase.wantErr)
			}
		})
	}
}
