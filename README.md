# mailer

[![Build Status](https://travis-ci.com/ViBiOh/mailer.svg?branch=master)](https://travis-ci.com/ViBiOh/mailer)
[![codecov](https://codecov.io/gh/ViBiOh/mailer/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/mailer)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/mailer)](https://goreportcard.com/report/github.com/ViBiOh/mailer)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=ViBiOh/mailer)](https://dependabot.com)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_mailer&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_mailer)

Mailer is a service for rendering and sending email based on Golang Template with the help of MJML and Mailjet.

# Getting Started

## Docker

Docker image is available, `vibioh/mailer` and a sample `docker-compose.yml`. Everything is almost configured, you only have to tweak domain's name, mainly configured for being used with [traefik](https://traefik.io), and adjust some secrets.

## Mailjet

In order to use the Mailjet sender, you need to register to [Mailjet](https://www.mailjet.com/) for having credentials.

## MJML

In order to use the MJML converter, you need to register to [MJML API](https://mjml.io/api) for having credentials or provided a compliant API like [mjml-api](https://github.com/ViBiOh/mjml-api).

## CLI Usage

```bash
Usage of mailer:
  -address string
        [http] Listen address {MAILER_ADDRESS}
  -cert string
        [http] Certificate file {MAILER_CERT}
  -corsCredentials
        [cors] Access-Control-Allow-Credentials {MAILER_CORS_CREDENTIALS}
  -corsExpose string
        [cors] Access-Control-Expose-Headers {MAILER_CORS_EXPOSE}
  -corsHeaders string
        [cors] Access-Control-Allow-Headers {MAILER_CORS_HEADERS} (default "Content-Type")
  -corsMethods string
        [cors] Access-Control-Allow-Methods {MAILER_CORS_METHODS} (default "GET")
  -corsOrigin string
        [cors] Access-Control-Allow-Origin {MAILER_CORS_ORIGIN} (default "*")
  -csp string
        [owasp] Content-Security-Policy {MAILER_CSP} (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options {MAILER_FRAME_OPTIONS} (default "deny")
  -hsts
        [owasp] Indicate Strict Transport Security {MAILER_HSTS} (default true)
  -key string
        [http] Key file {MAILER_KEY}
  -mailjetPrivateKey string
        [mailjet] Private Key {MAILER_MAILJET_PRIVATE_KEY}
  -mailjetPublicKey string
        [mailjet] Public Key {MAILER_MAILJET_PUBLIC_KEY}
  -mjmlPass string
        [mjml] Secret Key or Basic Auth pass {MAILER_MJML_PASS}
  -mjmlURL string
        [mjml] MJML API Converter URL {MAILER_MJML_URL} (default "https://api.mjml.io/v1/render")
  -mjmlUser string
        [mjml] Application ID or Basic Auth user {MAILER_MJML_USER}
  -okStatus int
        [http] Healthy HTTP Status code {MAILER_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {MAILER_PORT} (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics {MAILER_PROMETHEUS_PATH} (default "/metrics")
  -swaggerTitle string
        [swagger] API Title {MAILER_SWAGGER_TITLE} (default "API")
  -swaggerVersion string
        [swagger] API Version {MAILER_SWAGGER_VERSION} (default "1.0.0")
  -url string
        [alcotest] URL to check {MAILER_URL}
  -userAgent string
        [alcotest] User-Agent for check {MAILER_USER_AGENT} (default "Alcotest")
```
