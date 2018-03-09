default: go ui docker

go: deps dev

dev: format lint tst bench build

docker: docker-deps docker-build

deps:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u golang.org/x/tools/cmd/goimports
	dep ensure

format:
	goimports -w **/*.go *.go
	gofmt -s -w **/*.go *.go

lint:
	golint `go list ./... | grep -v vendor`
	errcheck -ignoretests `go list ./... | grep -v vendor`
	go vet ./...

tst:
	script/coverage

bench:
	go test ./... -bench . -benchmem -run Benchmark.*

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/mailer mailer.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -installsuffix nocgo -o bin/mailer-arm mailer.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -installsuffix nocgo -o bin/mailer-arm64 mailer.go

ui:
	npm run build

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-build:
	docker build -t ${DOCKER_USER}/mailer .
	docker build -t ${DOCKER_USER}/mailer -f Dockerifle_arm .
	docker build -t ${DOCKER_USER}/mailer -f Dockerifle_arm64 .

docker-push:
	docker login -u ${DOCKER_USER} -p ${DOCKER_PASS}
	docker push ${DOCKER_USER}/mailer
	docker push ${DOCKER_USER}/mailer:arm
	docker push ${DOCKER_USER}/mailer:arm64

start-deps:
	go get -u github.com/ViBiOh/auth/bcrypt

start-mailer:
	go run mailer.go \
		-authUsers "admin:admin" \
		-basicUsers "1:admin:`bcrypt admin`" \
		-directory "./dist"
