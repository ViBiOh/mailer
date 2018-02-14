package render

import (
	"net/http"

	"github.com/ViBiOh/httputils"
)

// App stores informations
type App struct {
}

// NewApp creates new App from Flags' config
func NewApp() *App {
	return &App{}
}

// Handler for Render request. Should be use with net/http
func (a *App) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(`Hello World !`)); err != nil {
			httputils.InternalServerError(w, err)
		}
	})
}
