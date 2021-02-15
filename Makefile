SHELL := /bin/bash

# go source files, ignore testdata directory
SRC = $(shell find . -type f -name '*.go' -not -path './testdata/*')

.PHONY: all fmt fmt-check vet check strict-check test test-race

all: test

fmt:
	gofmt -l -w $(SRC)

fmt-check:
	test -z $$(gofmt -l $(SRC))

vet:
	go vet ./...

check: vet

strict-check: check
	golint ./...

test: check
	go test ./...

test-race: check
	go test -race ./...
