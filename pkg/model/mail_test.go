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
	t.Parallel()

	cases := map[string]struct {
		instance MailRequest
		wantErr  error
	}{
		"empty": {
			NewMailRequest(),
			errors.New("from email is required"),
		},
		"no recipients": {
			NewMailRequest().From("nobody@localhost.fr"),
			errors.New("recipients are required"),
		},
		"empty recipients": {
			NewMailRequest().From("nobody@localhost.fr").To("john@doe.fr", "").To("john@john.fr"),
			errors.New("recipient at index 1 is empty"),
		},
		"empty template": {
			NewMailRequest().From("nobody@localhost.fr").To("john@doe.fr"),
			errors.New("template name is required"),
		},
		"valid": {
			NewMailRequest().From("nobody@localhost.fr").To("john@doe.fr").WithSubject("test").Template("test"),
			nil,
		},
	}

	for intention, testCase := range cases {
		intention, testCase := intention, testCase

		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			gotErr := testCase.instance.Check()

			failed := false

			if testCase.wantErr == nil && gotErr != nil {
				failed = true
			} else if testCase.wantErr != nil && gotErr == nil {
				failed = true
			} else if testCase.wantErr != nil && !strings.Contains(gotErr.Error(), testCase.wantErr.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("Check() = `%s`, want `%s`", gotErr, testCase.wantErr)
			}
		})
	}
}

func TestGetSubject(t *testing.T) {
	t.Parallel()

	type args struct {
		subject string
		payload any
	}

	cases := map[string]struct {
		args args
		want string
	}{
		"simple": {
			args{
				subject: "All is fine",
				payload: map[string]any{
					"Name": noop,
				},
			},
			"All is fine",
		},
		"invalid template": {
			args{
				subject: "All is fine {{-. toto}",
				payload: map[string]any{
					"Name": noop,
				},
			},
			"All is fine {{-. toto}",
		},
		"invalid exec": {
			args{
				subject: "All is fine Mr {{ .Name.Test }}",
				payload: map[string]any{
					"Name": noop,
				},
			},
			"All is fine Mr {{ .Name.Test }}",
		},
		"valid exec": {
			args{
				subject: "All is fine Mr {{ .Name }}",
				payload: map[string]any{
					"Name": "Test",
				},
			},
			"All is fine Mr Test",
		},
	}

	for intention, testCase := range cases {
		intention, testCase := intention, testCase

		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := getSubject(testCase.args.subject, testCase.args.payload); got != testCase.want {
				t.Errorf("getSubject() = `%s`, want `%s`", got, testCase.want)
			}
		})
	}
}
