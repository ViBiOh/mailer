package mailjet

import (
	"context"
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

var (
	// ErrNoConfiguration occurs when configuration is missing
	ErrNoConfiguration = errors.New(`No configuration for mailjet`)

	// ErrEmptyFrom occurs when from parameter is empty
	ErrEmptyFrom = errors.New(`"from" parameter is empty`)

	// ErrEmptyTo occurs when to parameter is empty
	ErrEmptyTo = errors.New(`"to" parameter is empty`)

	// ErrBlankTo occurs when a to recipient is blank
	ErrBlankTo = errors.New(`"to" item is blank`)
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

// App stores informations
type App struct {
	headers http.Header
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) *App {
	if *config[`publicKey`] == `` || *config[`privateKey`] == `` {
		return &App{}
	}

	return &App{
		headers: http.Header{`Authorization`: []string{request.GetBasicAuth(*config[`publicKey`], *config[`privateKey`])}},
	}
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`publicKey`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sPublicKey`, prefix)), ``, `Mailjet Public Key`),
		`privateKey`: flag.String(tools.ToCamel(fmt.Sprintf(`%sPrivateKey`, prefix)), ``, `Mailjet Private Key`),
	}
}

// CheckParameters checks mail descriptor
func (a App) CheckParameters(mail *Mail) error {
	if len(a.headers) == 0 {
		return ErrNoConfiguration
	}

	if strings.TrimSpace(mail.From) == `` {
		return ErrEmptyFrom
	}

	if len(mail.To) == 0 {
		return ErrEmptyTo
	}

	for _, to := range mail.To {
		if strings.TrimSpace(to.Email) == `` {
			return ErrBlankTo
		}
	}

	return nil
}

// GetParameters retrieves mail descriptor from Query
func (a App) GetParameters(r *http.Request) *Mail {
	mail := &Mail{
		From:    strings.TrimSpace(r.URL.Query().Get(`from`)),
		Sender:  strings.TrimSpace(r.URL.Query().Get(`sender`)),
		Subject: strings.TrimSpace(r.URL.Query().Get(`subject`)),
		To:      []Recipient{},
	}

	for _, rawTo := range strings.Split(r.URL.Query().Get(`to`), `,`) {
		if cleanTo := strings.TrimSpace(rawTo); cleanTo != `` {
			mail.To = append(mail.To, Recipient{Email: cleanTo})
		}
	}

	return mail
}

// SendMail send mailjet mail
func (a *App) SendMail(ctx context.Context, mail *Mail, html string) error {
	if err := a.CheckParameters(mail); err != nil {
		return nil
	}

	mail.HTML = html
	if payload, err := request.DoJSON(ctx, sendURL, mail, a.headers, http.MethodPost); err != nil {
		return fmt.Errorf(`Error while sending data: %v %s`, err, payload)
	}

	return nil
}

// Handler for Render request. Should be use with net/http
func (a App) Handler() http.Handler {
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

		if err := a.SendMail(r.Context(), a.GetParameters(r), string(content)); err != nil {
			if err == ErrEmptyFrom || err == ErrEmptyTo || err == ErrBlankTo {
				httperror.BadRequest(w, err)
			} else {
				httperror.InternalServerError(w, err)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
