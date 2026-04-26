.PHONY: all build test clean install-deps lint vet fmt cover

all: fmt vet lint test build

build:
	go build -o build/handsfree .

clean:
	rm -rf build coverage.out coverage.html

install-deps:
	go mod download

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./... -coverprofile=build/coverage.out
	go tool cover -html=build/coverage.out -o build/coverage.html

.DEFAULT_GOAL := all
