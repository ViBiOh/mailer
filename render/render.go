package render

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/httputils/httperror"
	"github.com/ViBiOh/httputils/request"
	"github.com/ViBiOh/httputils/templates"
)

// App stores informations
type App struct {
	tpl *template.Template
}

// NewApp creates new App from Flags' config
func NewApp() *App {
	return &App{
		tpl: template.Must(template.New(`mailer`).ParseGlob(`./templates/*.gohtml`)),
	}
}

// Handler for Render request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tpl := a.tpl.Lookup(fmt.Sprintf(`%s.gohtml`, strings.Trim(r.URL.Path, `/`)))

		if tpl == nil {
			httperror.NotFound(w)
			return
		}

		payload, err := request.ReadBody(r.Body)
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		var content map[string]interface{}
		if err := json.Unmarshal(payload, &content); err != nil {
			httperror.BadRequest(w, fmt.Errorf(`Error while unmarshalling payload %s: %v`, payload, err))
		}

		if err := templates.WriteHTMLTemplate(tpl, w, content, http.StatusOK); err != nil {
			httperror.InternalServerError(w, err)
		}
	})
}
