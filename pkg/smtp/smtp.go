package smtp

import (
	"flag"
	"net/smtp"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
)

// App of package
type App interface {
	Send(from string, to []string, content []byte) error
}

// Config of package
type Config struct {
	addr     *string
	user     *string
	password *string
	host     *string
}

type app struct {
	addr string
	auth smtp.Auth
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		addr:     flags.New(prefix, "smtp").Name("Address").Default("localhost:25").Label("Address").ToString(fs),
		user:     flags.New(prefix, "smtp").Name("AuthUser").Default("").Label("Plain Auth User").ToString(fs),
		password: flags.New(prefix, "smtp").Name("AuthPassword").Default("").Label("Plain Auth Password").ToString(fs),
		host:     flags.New(prefix, "smtp").Name("AuthHost").Default("localhost").Label("Plain Auth host").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	var auth smtp.Auth

	user := strings.TrimSpace(*config.user)
	if len(user) > 0 {
		auth = smtp.PlainAuth("", user, strings.TrimSpace(*config.password), strings.TrimSpace(*config.host))
	}

	return &app{
		addr: strings.TrimSpace(*config.addr),
		auth: auth,
	}
}

// Send emails to given recipients
func (a app) Send(from string, to []string, content []byte) error {
	return smtp.SendMail(a.addr, a.auth, from, to, content)
}
