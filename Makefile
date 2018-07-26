VERSION ?= $(shell git log --pretty=format:'%h' -n 1)
AUTHOR ?= $(shell git log --pretty=format:'%an' -n 1)
APP_NAME := mailer

default: api

api: deps go docker

go: format lint tst bench build

docker: docker-build docker-push

version:
	@echo -n $(VERSION)

author:
	@echo -n $(AUTHOR)

deps:
	go get github.com/golang/dep/cmd/dep
	go get github.com/golang/lint/golint
	go get github.com/kisielk/errcheck
	go get golang.org/x/tools/cmd/goimports
	dep ensure

format:
	goimports -w */*.go */*/*.go
	gofmt -s -w */*.go */*/*.go

lint:
	golint `go list ./... | grep -v vendor`
	errcheck -ignoretests `go list ./... | grep -v vendor`
	go vet ./...

tst:
	script/coverage

bench:
	go test ./... -bench . -benchmem -run Benchmark.*

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/$(APP_NAME) cmd/$(APP_NAME).go

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-login:
	echo $(DOCKER_PASS) | docker login -u $(DOCKER_USER) --password-stdin

docker-build: docker-deps
	docker build -t $(DOCKER_USER)/$(APP_NAME):$(VERSION) .

docker-push: docker-login
	docker push $(DOCKER_USER)/$(APP_NAME):$(VERSION)

docker-pull:
	docker pull $(DOCKER_USER)/$(APP_NAME):$(VERSION)

docker-promote: docker-pull
	docker tag $(DOCKER_USER)/$(APP_NAME):$(VERSION) $(DOCKER_USER)/$(APP_NAME):latest

docker-delete:
	curl -X DELETE -u "$(DOCKER_USER):$(DOCKER_CLOUD_TOKEN)" "https://cloud.docker.com/v2/repositories/$(DOCKER_USER)/$(APP_NAME)/tags/$(VERSION)/"

start-deps:
	go get github.com/ViBiOh/auth/cmd/bcrypt

start-api: start-deps
	go run cmd/$(APP_NAME).go \
		-tls=false \
		-authUsers "admin:admin" \
		-basicUsers "1:admin:`bcrypt admin`" \
		-csp "default-src 'self'; base-uri 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src fonts.gstatic.com; img-src 'self' http://i.imgur.com" \
    -mjmlURL $(MJML_URL) \
    -mjmlUser $(MJML_USER) \
    -mjmlPass $(MJML_PASS)

.PHONY: api go docker version author deps format lint tst bench build docker-deps docker-login docker-build docker-push docker-pull docker-promote docker-delete start-deps start-api
