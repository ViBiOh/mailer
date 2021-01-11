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
Usage of deploy:
  -address string
        [http] Listen address {DEPLOY_ADDRESS}
  -annotationPass string
        [annotation] Pass {DEPLOY_ANNOTATION_PASS}
  -annotationURL string
        [annotation] URL of Annotation server (e.g. my.grafana.com/api/annotations) {DEPLOY_ANNOTATION_URL}
  -annotationUser string
        [annotation] User {DEPLOY_ANNOTATION_USER}
  -apiNotification string
        [api] Email notificiation when deploy ends (possibles values ares 'never', 'onError', 'all') {DEPLOY_API_NOTIFICATION} (default "onError")
  -apiNotificationEmail string
        [api] Email address to notify {DEPLOY_API_NOTIFICATION_EMAIL}
  -apiTempFolder string
        [api] Temp folder for uploading files {DEPLOY_API_TEMP_FOLDER} (default "/tmp")
  -cert string
        [http] Certificate file {DEPLOY_CERT}
  -csp string
        [owasp] Content-Security-Policy {DEPLOY_CSP} (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options {DEPLOY_FRAME_OPTIONS} (default "deny")
  -graceDuration string
        [http] Grace duration when SIGTERM received {DEPLOY_GRACE_DURATION} (default "30s")
  -hsts
        [owasp] Indicate Strict Transport Security {DEPLOY_HSTS} (default true)
  -idleTimeout string
        [http] Idle Timeout {DEPLOY_IDLE_TIMEOUT} (default "2m")
  -key string
        [http] Key file {DEPLOY_KEY}
  -loggerJson
        [logger] Log format as JSON {DEPLOY_LOGGER_JSON}
  -loggerLevel string
        [logger] Logger level {DEPLOY_LOGGER_LEVEL} (default "INFO")
  -loggerLevelKey string
        [logger] Key for level in JSON {DEPLOY_LOGGER_LEVEL_KEY} (default "level")
  -loggerMessageKey string
        [logger] Key for message in JSON {DEPLOY_LOGGER_MESSAGE_KEY} (default "message")
  -loggerTimeKey string
        [logger] Key for timestamp in JSON {DEPLOY_LOGGER_TIME_KEY} (default "time")
  -mailerPass string
        [mailer] HTTP Pass {DEPLOY_MAILER_PASS}
  -mailerURL string
        [mailer] URL (https?:// or amqps?://) {DEPLOY_MAILER_URL}
  -mailerUser string
        [mailer] HTTP User {DEPLOY_MAILER_USER}
  -okStatus int
        [http] Healthy HTTP Status code {DEPLOY_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {DEPLOY_PORT} (default 1080)
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {DEPLOY_PROMETHEUS_IGNORE}
  -prometheusPath string
        [prometheus] Path for exposing metrics {DEPLOY_PROMETHEUS_PATH} (default "/metrics")
  -readTimeout string
        [http] Read Timeout {DEPLOY_READ_TIMEOUT} (default "5s")
  -shutdownTimeout string
        [http] Shutdown Timeout {DEPLOY_SHUTDOWN_TIMEOUT} (default "10s")
  -url string
        [alcotest] URL to check {DEPLOY_URL}
  -userAgent string
        [alcotest] User-Agent for check {DEPLOY_USER_AGENT} (default "Alcotest")
  -writeTimeout string
        [http] Write Timeout {DEPLOY_WRITE_TIMEOUT} (default "2m")
```
