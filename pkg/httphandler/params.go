package httphandler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ViBiOh/mailer/pkg/model"
)

var (
	errEmptyFrom = errors.New("\"from\" parameter is empty")
	errEmptyTo   = errors.New("\"to\" parameter is empty")
	errBlankTo   = errors.New("\"to\" item is blank")
)

func parseEmail(r *http.Request) model.Mail {
	email := model.Mail{
		From:    strings.TrimSpace(r.URL.Query().Get("from")),
		Sender:  strings.TrimSpace(r.URL.Query().Get("sender")),
		Subject: strings.TrimSpace(r.URL.Query().Get("subject")),
		To:      make([]string, 0),
	}

	for _, rawTo := range strings.Split(r.URL.Query().Get("to"), ",") {
		if cleanTo := strings.TrimSpace(rawTo); cleanTo != "" {
			email.To = append(email.To, cleanTo)
		}
	}

	return email
}

func checkEmail(email model.Mail) error {
	if strings.TrimSpace(email.From) == "" {
		return errEmptyFrom
	}

	if len(email.To) == 0 {
		return errEmptyTo
	}

	for _, to := range email.To {
		if strings.TrimSpace(to) == "" {
			return errBlankTo
		}
	}

	return nil
}
