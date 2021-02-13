package model

import "testing"

func noop() {
	// Do nothing
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
