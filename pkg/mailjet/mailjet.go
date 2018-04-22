package mailjet

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/tools"
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
func (a *App) SendMail(fromEmail, sender, subject string, to []string, html string) error {
	if len(a.headers) == 0 {
		return errors.New(`No configuration provided for Mailjet`)
	}

	recipients := make([]mailjetRecipient, 0, len(to))
	for _, rawTo := range to {
		recipients = append(recipients, mailjetRecipient{Email: rawTo})
	}

	mailjetMail := mailjetMail{FromEmail: fromEmail, FromName: sender, Subject: subject, Recipients: recipients, HTML: html}
	if payload, err := request.DoJSON(sendURL, mailjetMail, a.headers, http.MethodPost); err != nil {
		return fmt.Errorf(`Error while sending data: %v %s`, err, payload)
	}

	return nil
}

// SendFromRequest send mailjet mail
func (a *App) SendFromRequest(r *http.Request, html string) error {
	from := r.URL.Query().Get(`from`)
	sender := r.URL.Query().Get(`sender`)
	subject := r.URL.Query().Get(`subject`)
	to := strings.Split(r.URL.Query().Get(`to`), `,`)

	return a.SendMail(from, sender, subject, to, html)
}

// Handler for Render request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		content, err := request.ReadBody(r.Body)
		if err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`Error while reading request body: %v`, err))
			return
		}

		if err := a.SendFromRequest(r, string(content)); err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
