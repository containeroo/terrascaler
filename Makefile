VERSION ?= 0.1.0
IMG ?= ghcr.io/containeroo/terrascaler:latest

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTAINER_TOOL ?= docker

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy.
	go mod tidy

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.11.4
.PHONY: golangci-lint
golangci-lint: $(LOCALBIN) ## Download golangci-lint locally if necessary.
	@[ -f $(GOLANGCI_LINT) ] || { \
	set -e ;\
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell dirname $(GOLANGCI_LINT)) $(GOLANGCI_LINT_VERSION) ;\
	}

.PHONY: lint
lint: golangci-lint ## Run golangci-lint.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint and perform fixes.
	$(GOLANGCI_LINT) run --fix

##@ Build

.PHONY: build
build: fmt vet ## Build terrascaler binary.
	go build -ldflags="-s -w -X main.Version=v$(VERSION)" -o bin/terrascaler ./cmd/terrascaler

.PHONY: run
run: fmt vet ## Run terrascaler from your host.
	go run -ldflags="-X main.Version=dev" ./cmd/terrascaler

.PHONY: docker-build
docker-build: ## Build container image.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push container image.
	$(CONTAINER_TOOL) push ${IMG}

PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-buildx
docker-buildx: ## Build and push multi-arch container image.
	- $(CONTAINER_TOOL) buildx create --name terrascaler-builder
	$(CONTAINER_TOOL) buildx use terrascaler-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} .
	- $(CONTAINER_TOOL) buildx rm terrascaler-builder
