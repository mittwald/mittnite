GO ?= go
GIT ?= git
CGO_ENABLED ?= 0

VCS_VERSION = $(patsubst v%,%,$(shell $(GIT) describe --abbrev=0))
VCS_COMMIT  = $(shell $(GIT) rev-parse --short HEAD)
VCS_DATE    = $(shell date +%Y-%m-%dT%H:%M:%SZ%z)
GO_files = $(shell find . -type f -name '*.go' -and -not -name '*_test.go' -print)
GO_test_files = $(shell find . -type f -name '*_test.go' -and -not -name '*_integration_test.go' -print)
GO_integration_files = $(shell find . -type f -name '*_integration_test.go' -print)
GO_module = $(shell $(GO) list -m)
GOLD_FLAGS = -s -w \
             -extldflags "-static" \
             -X $(GO_module)/cmd.Version=$(VCS_VERSION) \
             -X $(GO_module)/cmd.Commit=$(VCS_COMMIT) \
             -X github.com/mittwald/mittnite/cmd.BuiltAt=$(VCS_DATE)
GOIT_FLAGS ?=

GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

ifeq ($(GOOS),windows)
	EXE_EXT := .exe
else
	EXE_EXT :=
endif

.PHONY: default
default: build

.PHONY: build
build: mittnite$(EXE_EXT)

.PHONY: lint
lint:
	$(GO) vet ./...

.PHONY: format
format:
	$(GO) fmt ./...

.PHONY: test
test:
	$(GO) test $(dir $(GO_test_files))

.PHONY: clean
clean:
	$(RM) mittnite$(EXE_EXT) $(wildcard mittnite-*)
	$(MAKE) -C test clean

.PHONY: integration-test
integration-test: integration-env
	$(GO) test -tags=integration \
		$(dir $(GO_integration_files)) \
		$(GOIT_FLAGS)

.PHONY: integration-clean
integration-clean:
	$(MAKE) -C test env-down

mittnite-$(GOOS)-$(GOARCH)$(EXE_EXT): $(GO_files)
	CGO_ENABLED=$(CGO_ENABLED) \
	GOOS=$(GOOS) GOARCH=$(GOARCH) \
	$(GO) build -trimpath -ldflags "$(GOLD_FLAGS)" -o $@

mittnite$(EXE_EXT): mittnite-$(GOOS)-$(GOARCH)$(EXE_EXT)
	cp $< $@

.PHONY: integration-env
integration-env:
	$(MAKE) -C test env-up
