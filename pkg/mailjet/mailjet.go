package mailjet

import (
	"context"
	"flag"
	"strings"

	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/request"
	"github.com/ViBiOh/mailer/pkg/model"
)

const (
	sendURL = "https://api.mailjet.com/v3/send"
)

// Recipient of an email
type Recipient struct {
	Email string `json:"Email"`
}

// Mail descriptor
type Mail struct {
	From    string      `json:"FromEmail"`
	Sender  string      `json:"FromName"`
	Subject string      `json:"Subject"`
	To      []Recipient `json:"Recipients"`
	HTML    string      `json:"Html-part"`
}

// Response from Mailjet
type Response struct {
	Sent []Recipient `json:"Sent"`
}

// Config of package
type Config struct {
	publicKey  *string
	privateKey *string
}

// App of package
type App interface {
	Send(ctx context.Context, mail model.Mail, html []byte) error
}

type app struct {
	publicKey  string
	privateKey string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		publicKey:  flags.New(prefix, "mailjet-api").Name("PublicKey").Default("").Label("Public Key").ToString(fs),
		privateKey: flags.New(prefix, "mailjet-api").Name("PrivateKey").Default("").Label("Private Key").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) App {
	publicKey := strings.TrimSpace(*config.publicKey)
	privateKey := strings.TrimSpace(*config.privateKey)

	if len(publicKey) == 0 || len(privateKey) == 0 {
		return nil
	}

	return app{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (a app) Send(ctx context.Context, mail model.Mail, html []byte) error {
	recipients := make([]Recipient, len(mail.To))
	for index, recipient := range mail.To {
		recipients[index] = Recipient{Email: recipient}
	}

	email := Mail{
		From:    mail.From,
		Sender:  mail.Sender,
		Subject: mail.Subject,
		To:      recipients,
		HTML:    string(html),
	}

	response, err := request.New().Post(sendURL).BasicAuth(a.publicKey, a.privateKey).JSON(ctx, email)
	if err != nil {
		return err
	}

	body, err := request.ReadBodyResponse(response)
	if err != nil {
		return err
	}

	logger.Info("Mail sent: %s", body)
	return nil
}
