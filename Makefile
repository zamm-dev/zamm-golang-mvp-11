.PHONY: build test clean install dev-setup lint fmt migrations-up

build:
	go build -o bin/zamm ./cmd/zamm

test:
	go test -race ./...

test-coverage:
	go tool cover -html=coverage.out -o coverage.html

update-golden:
	go test ./internal/cli/interactive/common/... -v -update

clean:
	rm -rf bin/ coverage.out coverage.html
	go clean -testcache

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