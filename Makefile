SHELL := /bin/bash
DOCKER_VERSION ?= $(shell git log --pretty=format:'%h' -n 1)
APP_NAME := mailer

default: api

api: deps go docker-api

ui: node docker-ui

go: format lint tst bench build

deps:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u golang.org/x/tools/cmd/goimports
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

node:
	npm run build

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-login:
	echo $(DOCKER_PASS) | docker login -u $(DOCKER_USER) --password-stdin

docker-pull: docker-pull-api docker-pull-ui

docker-promote: docker-pull docker-promote-api docker-promote-ui

docker-push: docker-push-api docker-push-ui

docker-api: docker-build-api docker-push-api

docker-build-api: docker-deps
	docker build -t $(DOCKER_USER)/$(APP_NAME)-api:$(DOCKER_VERSION) .

docker-push-api: docker-login
	docker push $(DOCKER_USER)/$(APP_NAME)-api:$(DOCKER_VERSION)

docker-pull-api:
	docker pull $(DOCKER_USER)/$(APP_NAME)-api:$(DOCKER_VERSION)

docker-promote-api:
	docker tag $(DOCKER_USER)/$(APP_NAME)-api:$(DOCKER_VERSION) $(DOCKER_USER)/$(APP_NAME)-api:latest

docker-ui: docker-build-ui docker-push-ui

docker-build-ui: docker-deps
	docker build -t $(DOCKER_USER)/$(APP_NAME)-ui:$(DOCKER_VERSION) -f ui/Dockerfile .

docker-push-ui: docker-login
	docker push $(DOCKER_USER)/$(APP_NAME)-ui:$(DOCKER_VERSION)

docker-pull-ui:
	docker pull $(DOCKER_USER)/$(APP_NAME)-ui:$(DOCKER_VERSION)

docker-promote-ui:
	docker tag $(DOCKER_USER)/$(APP_NAME)-ui:$(DOCKER_VERSION) $(DOCKER_USER)/$(APP_NAME)-ui:latest

start-deps:
	go get -u github.com/ViBiOh/auth/cmd/bcrypt

start-$(APP_NAME):
	go run cmd/$(APP_NAME).go \
		-authUsers "admin:admin" \
		-basicUsers "1:admin:`bcrypt admin`" \
		-directory "./dist"
