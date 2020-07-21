package client

import "testing"

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
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.Enabled(); got != tc.want {
				t.Errorf("Enabled() = %t, want %t", got, tc.want)
			}
		})
	}
}
