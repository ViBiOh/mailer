# mailer

[![Build](https://github.com/ViBiOh/mailer/workflows/Build/badge.svg)](https://github.com/ViBiOh/mailer/actions)
[![codecov](https://codecov.io/gh/ViBiOh/mailer/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/mailer)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/mailer)](https://goreportcard.com/report/github.com/ViBiOh/mailer)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_mailer&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_mailer)

Mailer is a service for rendering and sending email based on Golang Template with the help of MJML.

## Getting Started

Golang binary is built with static link. You can download it directly from the [Github Release page](https://github.com/ViBiOh/mailer/releases) or build it by yourself by cloning this repo and running `make`.

A Docker image is available for `amd64`, `arm` and `arm64` platforms on Docker Hub: [vibioh/mailer](https://hub.docker.com/r/vibioh/mailer/tags).

You can configure app by passing CLI args or environment variables (cf. [Usage](#usage) section). CLI override environment variables.

You'll find a Kubernetes exemple in the [`infra/`](infra/) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/master/app)

### MJML

In order to use the MJML converter, you need to register to [MJML API](https://mjml.io/api) for having credentials or provided a compliant API like [mjml-api](https://github.com/ViBiOh/mjml-api).

## Usage

```bash
Usage of mailer:
  -address string
        [http] Listen address {MAILER_ADDRESS}
  -amqpName string
        [amqp] Queue name {MAILER_AMQP_NAME} (default "mailer")
  -amqpURL string
        [amqp] Address {MAILER_AMQP_URL}
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
        [owasp] Content-Security-Policy {MAILER_CSP} (default "default-src 'self'; base-uri 'self'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' data: http://i.imgur.com grafana.com")
  -frameOptions string
        [owasp] X-Frame-Options {MAILER_FRAME_OPTIONS} (default "deny")
  -graceDuration string
        [http] Grace duration when SIGTERM received {MAILER_GRACE_DURATION} (default "30s")
  -hsts
        [owasp] Indicate Strict Transport Security {MAILER_HSTS} (default true)
  -idleTimeout string
        [http] Idle Timeout {MAILER_IDLE_TIMEOUT} (default "2m")
  -key string
        [http] Key file {MAILER_KEY}
  -loggerJson
        [logger] Log format as JSON {MAILER_LOGGER_JSON}
  -loggerLevel string
        [logger] Logger level {MAILER_LOGGER_LEVEL} (default "INFO")
  -loggerLevelKey string
        [logger] Key for level in JSON {MAILER_LOGGER_LEVEL_KEY} (default "level")
  -loggerMessageKey string
        [logger] Key for message in JSON {MAILER_LOGGER_MESSAGE_KEY} (default "message")
  -loggerTimeKey string
        [logger] Key for timestamp in JSON {MAILER_LOGGER_TIME_KEY} (default "time")
  -mailerTemplates string
        [mailer] Templates directory {MAILER_MAILER_TEMPLATES} (default "./templates/")
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
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {MAILER_PROMETHEUS_IGNORE}
  -prometheusPath string
        [prometheus] Path for exposing metrics {MAILER_PROMETHEUS_PATH} (default "/metrics")
  -readTimeout string
        [http] Read Timeout {MAILER_READ_TIMEOUT} (default "5s")
  -shutdownTimeout string
        [http] Shutdown Timeout {MAILER_SHUTDOWN_TIMEOUT} (default "10s")
  -smtpAddress string
        [smtp] Address {MAILER_SMTP_ADDRESS} (default "localhost:25")
  -smtpAuthHost string
        [smtp] Plain Auth host {MAILER_SMTP_AUTH_HOST} (default "localhost")
  -smtpAuthPassword string
        [smtp] Plain Auth Password {MAILER_SMTP_AUTH_PASSWORD}
  -smtpAuthUser string
        [smtp] Plain Auth User {MAILER_SMTP_AUTH_USER}
  -url string
        [alcotest] URL to check {MAILER_URL}
  -userAgent string
        [alcotest] User-Agent for check {MAILER_USER_AGENT} (default "Alcotest")
  -writeTimeout string
        [http] Write Timeout {MAILER_WRITE_TIMEOUT} (default "10s")
```
