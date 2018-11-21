SHELL := /bin/bash

# The name of the executable (default is current directory name)
TARGET := $(shell echo $${PWD\#\#*/})
.DEFAULT_GOAL: $(TARGET)

# These will be provided to the target
VERSION := 1.0.0
BUILD := `git rev-parse HEAD`

# go source files, ignore vendor directory
SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: all build clean install uninstall fmt simplify check run

all: test

$(TARGET): $(SRC)
	go build -o $(TARGET)

build: $(TARGET)
	@true

clean:
	rm -f $(TARGET)

fmt:
	gofmt -l -w $(SRC)

test: get-deps
	go test -short ./...

lint:
	go vet ./...

test-all: lint test
	go test -race ./...

strict-check:
	@test -z $(shell gofmt -l main.go | tee /dev/stderr) || echo "[WARN] Fix formatting issues with 'make fmt'"
	@for d in $$(go list ./... | grep -v /vendor/); do golint $${d}; done
	@go tool vet ${SRC}

get-deps:
	go get -t ./...

run: test-all install
	@$(TARGET)