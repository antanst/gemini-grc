SHELL := /usr/local/bin/oksh
export PATH := $(PATH)

all: fmt lintfix test clean build

clean:
	rm ./main

debug:
	@echo "PATH: $(PATH)"
	@echo "GOPATH: $(shell go env GOPATH)"
	@which go
	@which gofumpt
	@which gci
	@which golangci-lint

# Test
test:
	go test ./...

# Format code
fmt:
	gofumpt -l -w .
	gci write .

# Run linter
lint: fmt
	golangci-lint run

# Run linter and fix
lintfix: fmt
	golangci-lint run --fix

build:
	go build ./main.go
