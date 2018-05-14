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
	rm *.png

fmt:
	gofmt -l -w $(SRC)

test:
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
    dep init
    rm Gopkg.*
    rm -rv vendor

run: test-all install
	@$(TARGET)