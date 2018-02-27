package healthcheck

import (
	"net/http"

	"github.com/ViBiOh/mailer/mailjet"
	"github.com/ViBiOh/mailer/mjml"
)

// App stores informations
type App struct {
	mailjetApp *mailjet.App
	mjmlApp    *mjml.App
}

// NewApp creates new App from Flags' config
func NewApp(mailjetAppDep *mailjet.App, mjmlAppDep *mjml.App) *App {
	return &App{
		mailjetApp: mailjetAppDep,
		mjmlApp:    mjmlAppDep,
	}
}

// Handler for Healthcheck request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if a.mailjetApp.Ping() {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
