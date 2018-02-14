package mailjet

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/tools"
)

const (
	userURL = `https://api.mailjet.com/v3/user`
	sendURL = `https://api.mailjet.com/v3/send`
)

type mailjetRecipient struct {
	Email string `json:"Email"`
}

type mailjetMail struct {
	FromEmail  string             `json:"FromEmail"`
	FromName   string             `json:"FromName"`
	Subject    string             `json:"Subject"`
	Recipients []mailjetRecipient `json:"Recipients"`
	HTML       string             `json:"Html-part"`
}

type mailjetResponse struct {
	Sent []mailjetRecipient `json:"Sent"`
}

// App stores informations
type App struct {
	headers map[string]string
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) *App {
	if *config[`apiPublicKey`] == `` {
		return &App{}
	}

	return &App{
		headers: map[string]string{`Authorization`: httputils.GetBasicAuth(*config[`apiPublicKey`], *config[`apiPrivateKey`])},
	}
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`apiPublicKey`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sMailjetPublicKey`, prefix)), ``, `Mailjet Public Key`),
		`apiPrivateKey`: flag.String(tools.ToCamel(fmt.Sprintf(`%sMailjetPrivateKey`, prefix)), ``, `Mailjet Private Key`),
	}
}

// Ping indicate if Mailjet is ready or not
func (a *App) Ping() bool {
	if a.headers != nil {
		return true
	}

	if _, err := httputils.Request(userURL, nil, a.headers, http.MethodGet); err != nil {
		log.Printf(`[mailjet] Error while pinging: %v`, err)
		return false
	}
	return true
}

// SendMail send mailjet mail
func (a *App) SendMail(fromEmail string, fromName string, subject string, to []string, html string) error {
	recipients := make([]mailjetRecipient, 0, len(to))
	for _, rawTo := range to {
		recipients = append(recipients, mailjetRecipient{Email: rawTo})
	}

	mailjetMail := mailjetMail{FromEmail: fromEmail, FromName: fromName, Subject: subject, Recipients: recipients, HTML: html}
	if _, err := httputils.RequestJSON(sendURL, mailjetMail, a.headers, http.MethodPost); err != nil {
		return fmt.Errorf(`Error while sending data to %s: %v`, sendURL, err)
	}

	return nil
}
