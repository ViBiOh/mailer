package model

import (
	"errors"
	"strings"
	"testing"
)

func noop() {
	// Do nothing
}

func TestCheck(t *testing.T) {
	var cases = []struct {
		intention string
		instance  MailRequest
		wantErr   error
	}{
		{
			"empty",
			NewMailRequest(),
			errors.New("from email is required"),
		},
		{
			"no recipients",
			NewMailRequest().From("nobody@localhost.fr"),
			errors.New("recipients are required"),
		},
		{
			"empty recipients",
			NewMailRequest().From("nobody@localhost.fr").To("john@doe.fr", "").To("john@john.fr"),
			errors.New("recipient at index 1 is empty"),
		},
		{
			"empty template",
			NewMailRequest().From("nobody@localhost.fr").To("john@doe.fr"),
			errors.New("template name is required"),
		},
		{
			"valid",
			NewMailRequest().From("nobody@localhost.fr").To("john@doe.fr").WithSubject("test").Template("test"),
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			gotErr := tc.instance.Check()

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("Check() = `%s`, want `%s`", gotErr, tc.wantErr)
			}
		})
	}
}

func TestGetSubject(t *testing.T) {
	type args struct {
		subject string
		payload interface{}
	}

	var cases = []struct {
		intention string
		args      args
		want      string
	}{
		{
			"simple",
			args{
				subject: "All is fine",
				payload: map[string]interface{}{
					"Name": noop,
				},
			},
			"All is fine",
		},
		{
			"invalid template",
			args{
				subject: "All is fine {{-. toto}",
				payload: map[string]interface{}{
					"Name": noop,
				},
			},
			"All is fine {{-. toto}",
		},
		{
			"invalid exec",
			args{
				subject: "All is fine Mr {{ .Name.Test }}",
				payload: map[string]interface{}{
					"Name": noop,
				},
			},
			"All is fine Mr {{ .Name.Test }}",
		},
		{
			"valid exec",
			args{
				subject: "All is fine Mr {{ .Name }}",
				payload: map[string]interface{}{
					"Name": "Test",
				},
			},
			"All is fine Mr Test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := getSubject(tc.args.subject, tc.args.payload); got != tc.want {
				t.Errorf("getSubject() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
