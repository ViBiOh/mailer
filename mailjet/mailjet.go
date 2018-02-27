package mailjet

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/ViBiOh/httputils/request"
	"github.com/ViBiOh/httputils/tools"
)

const (
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
	if *config[`publicKey`] == `` {
		return &App{}
	}

	return &App{
		headers: map[string]string{`Authorization`: request.GetBasicAuth(*config[`publicKey`], *config[`privateKey`])},
	}
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`publicKey`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sPublicKey`, prefix)), ``, `Mailjet Public Key`),
		`privateKey`: flag.String(tools.ToCamel(fmt.Sprintf(`%sPrivateKey`, prefix)), ``, `Mailjet Private Key`),
	}
}

// SendMail send mailjet mail
func (a *App) SendMail(fromEmail string, fromName string, subject string, to []string, html string) error {
	recipients := make([]mailjetRecipient, 0, len(to))
	for _, rawTo := range to {
		recipients = append(recipients, mailjetRecipient{Email: rawTo})
	}

	mailjetMail := mailjetMail{FromEmail: fromEmail, FromName: fromName, Subject: subject, Recipients: recipients, HTML: html}
	if payload, err := request.DoJSON(sendURL, mailjetMail, a.headers, http.MethodPost); err != nil {
		return fmt.Errorf(`Error while sending data: %v %s`, err, payload)
	}

	return nil
}
