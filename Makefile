# Makefile for the `unleash-client-go` project

# tools
GO_BIN_NAME := go
GO_BIN := $(shell command -v $(GO_BIN_NAME) 2> /dev/null)
EXTRA_PATH=$(shell dirname $(GO_BINDATA_BIN))
DEP_BIN_NAME := dep
DEP_BIN := $(shell command -v $(DEP_BIN_NAME) 2> /dev/null)
VENDOR_DIR=vendor

SOURCE_DIR ?= .
SOURCES := $(shell find $(SOURCE_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)

# Call this function with $(call log-info,"Your message")
define log-info =
@echo "INFO: $(1)"
endef


.PHONY: help
# Based on https://gist.github.com/rcmachado/af3db315e31383502660
## Display this help text.
help:/
	$(info Available targets)
	$(info -----------------)
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		helpCommand = substr($$1, 0, index($$1, ":")-1); \
		if (helpMessage) { \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub(/##/, "\n                                     ", helpMessage); \
		} else { \
			helpMessage = "(No documentation)"; \
		} \
		printf "%-35s - %s\n", helpCommand, helpMessage; \
		lastLine = "" \
	} \
	{ hasComment = match(lastLine, /^## (.*)/); \
          if(hasComment) { \
            lastLine=lastLine$$0; \
	  } \
          else { \
	    lastLine = $$0 \
          } \
        }' $(MAKEFILE_LIST)

.PHONY: prebuild-checks
prebuild-checks: $(TMP_PATH) $(INSTALL_PREFIX) 
# Check that all tools where found
ifndef DEP_BIN
	$(error The "$(DEP_BIN_NAME)" executable could not be found in your PATH)
endif
ifndef GO_BIN
	$(error The "$(GO_BIN_NAME)" executable could not be found in your PATH)
endif


.PHONY: deps 
## Download build dependencies.
deps: $(VENDOR_DIR) 

$(VENDOR_DIR): 
	@echo "verifying deps..."
	$(DEP_BIN) ensure -v

.PHONY: clean
## cleans the vendor directory
clean:
	@rm -rf vendor
	
.PHONY: vet
## runs the 'go vet' command
vet:
	@go vet ./...

.PHONY: test
## run all tests except in the 'vendor' package 
test: build vet
	@echo "running tests..."
	@go test -v ./... 

.PHONY: build
## run all tests except in the 'vendor' package 
build: deps
	@echo "checking that the client library can build..."
	@go build *.go