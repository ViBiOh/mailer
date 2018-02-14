# mailer

[![Build Status](https://travis-ci.org/ViBiOh/mailer.svg?branch=master)](https://travis-ci.org/ViBiOh/mailer)
[![codecov](https://codecov.io/gh/ViBiOh/mailer/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/mailer)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/mailer)](https://goreportcard.com/report/github.com/ViBiOh/mailer)

## Usage

```
Usage of mailer:
  -c string
      [health] URL to check
  -corsCredentials
      [cors] Access-Control-Allow-Credentials
  -corsExpose string
      [cors] Access-Control-Expose-Headers
  -corsHeaders string
      [cors] Access-Control-Allow-Headers (default "Content-Type")
  -corsMethods string
      [cors] Access-Control-Allow-Methods (default "GET")
  -corsOrigin string
      [cors] Access-Control-Allow-Origin (default "*")
  -csp string
      [owasp] Content-Security-Policy (default "default-src 'self'")
  -frameOptions string
      [owasp] X-Frame-Options (default "deny")
  -hsts
      [owasp] Indicate Strict Transport Security (default true)
  -port string
      Listen port (default "1080")
  -tls
      Serve TLS content
  -tlsCert string
      [tls] PEM Certificate file
  -tlsHosts string
      [tls] Self-signed certificate hosts, comma separated (default "localhost")
  -tlsKey string
      [tls] PEM Key file
```