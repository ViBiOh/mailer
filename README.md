# mailer

[![Build Status](https://travis-ci.com/ViBiOh/mailer.svg?branch=master)](https://travis-ci.com/ViBiOh/mailer)
[![codecov](https://codecov.io/gh/ViBiOh/mailer/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/mailer)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/mailer)](https://goreportcard.com/report/github.com/ViBiOh/mailer)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=ViBiOh/mailer)](https://dependabot.com)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_mailer&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_mailer)

Mailer is a service for rendering and sending email based on Golang Template with the help of MJML.

## Getting Started

Golang binary is built with static link. You can download it directly from the [Github Release page](https://github.com/ViBiOh/mailer/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/mailer](https://hub.docker.com/r/vibioh/mailer/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple (without secrets) in the [`infra/`](infra/) folder.

### MJML

In order to use the MJML converter, you need to register to [MJML API](https://mjml.io/api) for having credentials or provided a compliant API like [mjml-api](https://github.com/ViBiOh/mjml-api).

## Usage

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
  -graceDuration string
        [http] Grace duration when SIGTERM received {MAILER_GRACE_DURATION} (default "15s")
  -hsts
        [owasp] Indicate Strict Transport Security {MAILER_HSTS} (default true)
  -key string
        [http] Key file {MAILER_KEY}
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
  -smtpAddress string
        [smtp] Address {MAILER_SMTP_ADDRESS} (default "localhost:25")
  -smtpAuthHost string
        [smtp] Plain Auth host {MAILER_SMTP_AUTH_HOST}
  -smtpAuthPassword string
        [smtp] Plain Auth Password {MAILER_SMTP_AUTH_PASSWORD}
  -smtpAuthUser string
        [smtp] Plain Auth User {MAILER_SMTP_AUTH_USER}
  -swaggerTitle string
        [swagger] API Title {MAILER_SWAGGER_TITLE} (default "API")
  -swaggerVersion string
        [swagger] API Version {MAILER_SWAGGER_VERSION} (default "1.0.0")
  -url string
        [alcotest] URL to check {MAILER_URL}
  -userAgent string
        [alcotest] User-Agent for check {MAILER_USER_AGENT} (default "Alcotest")
```
