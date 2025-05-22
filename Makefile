SHELL := /bin/sh
export PATH := $(PATH)

all: fmt lintfix vet tidy test clean build

clean:
	mkdir -p ./dist && rm -rf ./dist/*

debug:
	@echo "PATH: $(PATH)"
	@echo "GOPATH: $(shell go env GOPATH)"
	@which go
	@which gofumpt
	@which gci
	@which golangci-lint

# Test
test:
	go test -race ./...

tidy:
	go mod tidy

# Format code
fmt:
	gofumpt -l -w .
	gci write .

# Run linter
lint: fmt
	golangci-lint run

vet: fmt
	go vet ./.../

# Run linter and fix
lintfix: fmt
	golangci-lint run --fix

build:
	CGO_ENABLED=0 go build -o ./dist/get ./cmd/get/get.go
	CGO_ENABLED=0 go build -o ./dist/crawl ./cmd/crawl/crawl.go
	CGO_ENABLED=0 go build -o ./dist/crawler ./cmd/crawler/crawler.go

show-updates:
	go list -m -u all

update:
	go get -u all

update-patch:
	go get -u=patch all
