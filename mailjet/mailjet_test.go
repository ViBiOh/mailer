package mailjet

import (
	"testing"
)

func TestPing(t *testing.T) {
	emptyValue := ``

	var cases = []struct {
		apiPublicKey string
		want         bool
	}{
		{
			``,
			false,
		},
		{
			`test`,
			true,
		},
	}

	for _, testCase := range cases {
		app := NewApp(map[string]*string{`apiPublicKey`: &testCase.apiPublicKey, `apiPrivateKey`: &emptyValue})

		if result := app.Ping(); result != testCase.want {
			t.Errorf(`Ping() = %v, want %v, with apiPublicKey=%v`, result, testCase.want, testCase.apiPublicKey)
		}
	}
}
