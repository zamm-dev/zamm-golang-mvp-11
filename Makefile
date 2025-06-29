.PHONY: build test clean install dev-setup lint fmt migrations-up

build:
	go build -o bin/zamm ./cmd/zamm

test:
	go test -v -race -coverprofile=coverage.out ./...

test-coverage:
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/ coverage.out coverage.html

install:
	go install ./cmd/zamm

dev-setup:
	go mod download
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	golangci-lint run

fmt:
	go fmt ./...

migrations-up:
	./bin/zamm migration up