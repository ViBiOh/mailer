SHELL := /bin/bash
DOCKER_VERSION ?= $(shell git log --pretty=format:'%h' -n 1)

default: go

go: deps api docker-build-api docker-push-api

api: format lint tst bench build

ui: node docker-build-ui docker-push-ui

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
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/mailer cmd/mailer.go

node:
	npm run build

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-login:
	echo $(DOCKER_PASS) | docker login -u $(DOCKER_USER) --password-stdin

docker-pull: docker-pull-api docker-pull-notifier docker-pull-ui

docker-promote: docker-pull docker-promote-api docker-promote-ui

docker-push: docker-push-api docker-push-ui

docker-build-api: docker-deps
	docker build -t $(DOCKER_USER)/mailer-api:$(DOCKER_VERSION) .

docker-push-api: docker-login
	docker push $(DOCKER_USER)/mailer-api:$(DOCKER_VERSION)

docker-pull-api:
	docker pull $(DOCKER_USER)/mailer-api:$(DOCKER_VERSION)

docker-promote-api:
	docker tag $(DOCKER_USER)/mailer-api:$(DOCKER_VERSION) $(DOCKER_USER)/mailer-api:latest

docker-build-ui: docker-deps
	docker build -t $(DOCKER_USER)/mailer-ui:$(DOCKER_VERSION) -f ui/Dockerfile .

docker-push-ui: docker-login
	docker push $(DOCKER_USER)/mailer-ui:$(DOCKER_VERSION)

docker-pull-api:
	docker pull $(DOCKER_USER)/mailer-ui:$(DOCKER_VERSION)

docker-promote-ui:
	docker tag $(DOCKER_USER)/mailer-ui:$(DOCKER_VERSION) $(DOCKER_USER)/mailer-ui:latest

start-deps:
	go get -u github.com/ViBiOh/auth/cmd/bcrypt

start-mailer:
	go run cmd/mailer.go \
		-authUsers "admin:admin" \
		-basicUsers "1:admin:`bcrypt admin`" \
		-directory "./dist"
