package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/ViBiOh/alcotest/alcotest"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cert"
	"github.com/ViBiOh/httputils/cors"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/mailer/healthcheck"
	"github.com/ViBiOh/mailer/mailjet"
	"github.com/ViBiOh/mailer/render"
)

const (
	healthcheckPath = `/health`
	mailPath        = `/mail`
)

var (
	healthcheckHandler http.Handler
	renderHandler      http.Handler
)

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, healthcheckPath) {
			healthcheckHandler.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, mailPath) {
			renderHandler.ServeHTTP(w, r)
		} else {
			httputils.NotFound(w)
		}
	})
}

func main() {
	port := flag.String(`port`, `1080`, `Listen port`)
	tls := flag.Bool(`tls`, false, `Serve TLS content`)

	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	corsConfig := cors.Flags(`cors`)
	owaspConfig := owasp.Flags(``)

	mailjetConfig := mailjet.Flags(``)

	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	log.Printf(`Starting server on port %s`, *port)

	mailjetApp := mailjet.NewApp(mailjetConfig)

	renderApp := render.NewApp()
	renderHandler = http.StripPrefix(mailPath, renderApp.Handler())

	healthcheckApp := healthcheck.NewApp(mailjetApp)
	healthcheckHandler = http.StripPrefix(healthcheckPath, healthcheckApp.Handler())

	server := &http.Server{
		Addr:    `:` + *port,
		Handler: owasp.Handler(owaspConfig, cors.Handler(corsConfig, handler())),
	}

	var serveError = make(chan error)
	go func() {
		defer close(serveError)
		if *tls {
			log.Print(`Listening with TLS enabled`)
			serveError <- cert.ListenAndServeTLS(certConfig, server)
		} else {
			serveError <- server.ListenAndServe()
		}
	}()

	httputils.ServerGracefulClose(server, serveError, nil)
}
