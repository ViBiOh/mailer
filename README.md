# mailer

[![Build Status](https://travis-ci.org/ViBiOh/mailer.svg?branch=master)](https://travis-ci.org/ViBiOh/mailer)
[![codecov](https://codecov.io/gh/ViBiOh/mailer/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/mailer)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/mailer)](https://goreportcard.com/report/github.com/ViBiOh/mailer)

Mailer is a service for rendering and sending email based on Golang Template with the help of MJML and Mailjet.

# Getting Started

## Docker

Docker image is available, `vibioh/mailer` and a sample `docker-compose.yml`. Everything is almost configured, you only have to tweak domain's name, mainly configured for being used with [traefik](https://traefik.io), and adjust some secrets.

## Mailjet

In order to use the Mailjet sender, you need to register to [Mailjet](https://www.mailjet.com/) for having credentials.

## MJML

In order to use the MJML converter, you need to register to [MJML API](https://mjml.io/api) for having credentials or provided a compliant API like [mjml-api](https://github.com/ViBiOh/mjml-api).

# Build

## Usage

```bash
Usage of mailer:
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
        [owasp] Content-Security-Policy (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options (default "deny")
  -hsts
        [owasp] Indicate Strict Transport Security (default true)
  -mailjetPrivateKey string
        [mailjet] Private Key
  -mailjetPublicKey string
        [mailjet] Public Key
  -mjmlPass string
        [mjml] Secret Key or Basic Auth pass
  -mjmlURL string
        [mjml] MJML API Converter URL (default "https://api.mjml.io/v1/render")
  -mjmlUser string
        [mjml] Application ID or Basic Auth user
  -port int
        Listen port (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics (default "/metrics")
  -tls
        Serve TLS content (default true)
  -tlsCert string
        [tls] PEM Certificate file
  -tlsHosts string
        [tls] Self-signed certificate hosts, comma separated (default "localhost")
  -tlsKey string
        [tls] PEM Key file
  -tlsOrganization string
        [tls] Self-signed certificate organization (default "ViBiOh")
  -tracingAgent string
        [opentracing] Jaeger Agent (e.g. host:port) (default "jaeger:6831")
  -tracingName string
        [opentracing] Service name
  -url string
        [health] URL to check
  -userAgent string
        [health] User-Agent for check (default "Golang alcotest")
```