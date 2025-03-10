SHELL := /bin/env oksh
export PATH := $(PATH)

all: fmt lintfix tidy test clean build

clean:
	rm -f ./dist && mkdir ./dist

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

# Run linter and fix
lintfix: fmt
	golangci-lint run --fix

build:
	CGO_ENABLED=0 go build -o ./dist/gemini-grc ./main.go

show-updates:
	go list -m -u all

update:
	go get -u all

update-patch:
	go get -u=patch all
