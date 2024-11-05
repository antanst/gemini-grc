SHELL := /usr/local/bin/oksh
export PATH := $(PATH)

.PHONY: all fmt lint

all: fmt lint test

# Test
test:
	go test -v ./...

# Format code
fmt:
	gofumpt -l -w .
	gci write .

# Run linter
lint: fmt
	golangci-lint run
