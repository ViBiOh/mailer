package mailjet

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/tools"
)

const (
	sendURL = "https://api.mailjet.com/v3/send"
)

var (
	// ErrNoConfiguration occurs when configuration is missing
	ErrNoConfiguration = errors.New("no configuration for mailjet")

	// ErrEmptyFrom occurs when from parameter is empty
	ErrEmptyFrom = errors.New("\"from\" parameter is empty")

	// ErrEmptyTo occurs when to parameter is empty
	ErrEmptyTo = errors.New("\"to\" parameter is empty")

	// ErrBlankTo occurs when a to recipient is blank
	ErrBlankTo = errors.New("\"to\" item is blank")
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
type App struct {
	headers http.Header
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		publicKey:  fs.String(tools.ToCamel(fmt.Sprintf("%sPublicKey", prefix)), "", "[mailjet] Public Key"),
		privateKey: fs.String(tools.ToCamel(fmt.Sprintf("%sPrivateKey", prefix)), "", "[mailjet] Private Key"),
	}
}

// New creates new App from Config
func New(config Config) *App {
	publicKey := strings.TrimSpace(*config.publicKey)
	privateKey := strings.TrimSpace(*config.privateKey)

	if publicKey == "" || privateKey == "" {
		return &App{}
	}

	return &App{
		headers: http.Header{"Authorization": []string{request.GenerateBasicAuth(publicKey, privateKey)}},
	}
}

// CheckParameters checks mail descriptor
func (a App) CheckParameters(mail *Mail) error {
	if len(a.headers) == 0 {
		return ErrNoConfiguration
	}

	if strings.TrimSpace(mail.From) == "" {
		return ErrEmptyFrom
	}

	if len(mail.To) == 0 {
		return ErrEmptyTo
	}

	for _, to := range mail.To {
		if strings.TrimSpace(to.Email) == "" {
			return ErrBlankTo
		}
	}

	return nil
}

// GetParameters retrieves mail descriptor from Query
func (a App) GetParameters(r *http.Request) *Mail {
	mail := &Mail{
		From:    strings.TrimSpace(r.URL.Query().Get("from")),
		Sender:  strings.TrimSpace(r.URL.Query().Get("sender")),
		Subject: strings.TrimSpace(r.URL.Query().Get("subject")),
		To:      []Recipient{},
	}

	for _, rawTo := range strings.Split(r.URL.Query().Get("to"), ",") {
		if cleanTo := strings.TrimSpace(rawTo); cleanTo != "" {
			mail.To = append(mail.To, Recipient{Email: cleanTo})
		}
	}

	return mail
}

// SendMail send mailjet mail
func (a App) SendMail(ctx context.Context, mail *Mail, html string) error {
	if err := a.CheckParameters(mail); err != nil {
		return err
	}

	mail.HTML = html
	if _, _, _, err := request.DoJSON(ctx, sendURL, mail, a.headers, http.MethodPost); err != nil {
		return err
	}

	return nil
}
