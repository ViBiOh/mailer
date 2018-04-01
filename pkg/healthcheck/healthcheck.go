package healthcheck

import (
	"net/http"

	"github.com/ViBiOh/mailer/pkg/mailjet"
)

// App stores informations
type App struct {
	mailjetApp *mailjet.App
}

// NewApp creates new App from Flags' config
func NewApp(mailjetAppDep *mailjet.App) *App {
	return &App{
		mailjetApp: mailjetAppDep,
	}
}

// Handler for Healthcheck request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
