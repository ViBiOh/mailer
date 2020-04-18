package render

import (
	"errors"
	"net/http"
	"strings"
)

var (
	errEmptyFrom = errors.New("\"from\" parameter is empty")
	errEmptyTo   = errors.New("\"to\" parameter is empty")
	errBlankTo   = errors.New("\"to\" item is blank")
)

type email struct {
	from    string
	sender  string
	to      []string
	subject string
}

func parseEmail(r *http.Request) email {
	email := email{
		from:    strings.TrimSpace(r.URL.Query().Get("from")),
		sender:  strings.TrimSpace(r.URL.Query().Get("sender")),
		subject: strings.TrimSpace(r.URL.Query().Get("subject")),
		to:      make([]string, 0),
	}

	for _, rawTo := range strings.Split(r.URL.Query().Get("to"), ",") {
		if cleanTo := strings.TrimSpace(rawTo); cleanTo != "" {
			email.to = append(email.to, cleanTo)
		}
	}

	return email
}

func checkEmail(values email) error {
	if strings.TrimSpace(values.from) == "" {
		return errEmptyFrom
	}

	if len(values.to) == 0 {
		return errEmptyTo
	}

	for _, to := range values.to {
		if strings.TrimSpace(to) == "" {
			return errBlankTo
		}
	}

	return nil
}
