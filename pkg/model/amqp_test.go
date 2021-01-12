package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/streadway/amqp"
)

func TestEnabled(t *testing.T) {
	var cases = []struct {
		intention string
		instance  AMQPClient
		want      bool
	}{
		{
			"empty",
			AMQPClient{},
			false,
		},
		{
			"connection",
			AMQPClient{
				connection: &amqp.Connection{},
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

func TestPing(t *testing.T) {
	var cases = []struct {
		intention string
		instance  AMQPClient
		want      error
	}{
		{
			"empty",
			AMQPClient{},
			errors.New("amqp client closed"),
		},
		{
			"not opened",
			AMQPClient{
				connection: &amqp.Connection{},
			},
			errors.New("amqp client closed"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			got := tc.instance.Ping()

			failed := false

			if tc.want == nil && got != nil {
				failed = true
			} else if tc.want != nil && got == nil {
				failed = true
			} else if tc.want != nil && !strings.Contains(got.Error(), tc.want.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("Ping() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
