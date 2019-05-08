SHELL = /bin/sh

APP_NAME ?= mailer

PACKAGES ?= ./...
APP_PACKAGES = $(shell go list -e $(PACKAGES) | grep -v node_modules)

GOBIN=bin
BINARY_PATH=$(GOBIN)/$(APP_NAME)

SERVER_SOURCE = cmd/$(APP_NAME).go
SERVER_RUNNER = go run $(SERVER_SOURCE)
ifeq ($(DEBUG), true)
	SERVER_RUNNER = dlv debug $(SERVER_SOURCE) --
endif

.PHONY: help
help: Makefile
	@sed -n 's|^##||p' $< | column -t -s ':' | sed -e 's|^| |'

## $(APP_NAME): Build app with dependencies download
.PHONY: $(APP_NAME)
$(APP_NAME): deps go

.PHONY: go
go: format lint test bench build

## name: Output name of app
.PHONY: name
name:
	@echo -n $(APP_NAME)

## dist: Output build output path
.PHONY: dist
dist:
	@echo -n $(BINARY_PATH)

## version: Output sha1 of last commit
.PHONY: version
version:
	@echo -n $(shell git rev-parse --short HEAD)

## author: Output author's name of last commit
.PHONY: author
author:
	@python -c 'import sys; import urllib; sys.stdout.write(urllib.quote_plus(sys.argv[1]))' "$(shell git log --pretty=format:'%an' -n 1)"

## deps: Download dependencies
.PHONY: deps
deps:
	go get github.com/kisielk/errcheck
	go get golang.org/x/lint/golint
	go get golang.org/x/tools/cmd/goimports

## format: Format code of app
.PHONY: format
format:
	goimports -w */*.go */*/*.go
	gofmt -s -w */*.go */*/*.go

## lint: Lint code of app
.PHONY: lint
lint:
	golint $(APP_PACKAGES)
	errcheck -ignoretests $(APP_PACKAGES)
	go vet $(APP_PACKAGES)

## test: Test code of app with coverage
.PHONY: test
test:
	script/coverage

## bench: Benchmark code of app
.PHONY: bench
bench:
	go test $(APP_PACKAGES) -bench . -benchmem -run Benchmark.*

## build: Build binary of app
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o $(BINARY_PATH) $(SERVER_SOURCE)

## start: Start app
.PHONY: start
start:
	$(SERVER_RUNNER) \
		-csp "default-src 'self'; base-uri 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' http://i.imgur.com" \
		-mjmlURL $(MJML_URL) \
		-mjmlUser $(MJML_USER) \
		-mjmlPass $(MJML_PASS)
