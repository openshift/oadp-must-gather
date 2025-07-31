# Makefile for OADP Must-gather

# Matches bin name from the Dockerfile
# Project Variables
BINARY_NAME = gather
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_REV := $(shell git rev-parse --short HEAD)

# Container Variables
CONTAINER_TOOL ?= $(shell \
  if command -v docker >/dev/null 2>&1; then echo docker; \
  elif command -v podman >/dev/null 2>&1; then echo podman; \
  else echo ""; \
  fi \
)
ifeq ($(CONTAINER_TOOL),)
  $(error No supported container tool (docker or podman) found in PATH. Please install one.)
endif

# Image Configuration
IMG ?= ttl.sh/oadp-must-gather-$(GIT_REV):1h
CONTAINERFILE ?= Dockerfile

# Platform Configuration for multi-arch builds
PLATFORMS = linux/amd64 linux/arm64 linux/ppc64le linux/s390x
PLATFORM ?=
GOOS = $(word 1,$(subst /, ,$(PLATFORM)))
GOARCH = $(word 2,$(subst /, ,$(PLATFORM)))

# Build arguments for container platform matching
OC_CLI ?= $(shell which oc)
CLUSTER_TYPE_SHELL := $(shell $(OC_CLI) get infrastructures cluster -o jsonpath='{.status.platform}' 2> /dev/null | tr A-Z a-z)
CLUSTER_TYPE ?= $(CLUSTER_TYPE_SHELL)
CLUSTER_OS = $(shell $(OC_CLI) get node -o jsonpath='{.items[0].status.nodeInfo.operatingSystem}' 2> /dev/null)
CLUSTER_ARCH = $(shell $(OC_CLI) get node -o jsonpath='{.items[0].status.nodeInfo.architecture}' 2> /dev/null)

CONTAINER_BUILD_ARGS ?= --platform=linux/amd64
ifneq ($(CLUSTER_TYPE),)
	CONTAINER_BUILD_ARGS = --platform=$(CLUSTER_OS)/$(CLUSTER_ARCH)
endif

# Go build flags
GO_BUILD_FLAGS ?= -mod=mod
GO_TEST_FLAGS ?= -mod=mod

# Use bash if available, else fall back to sh
SHELL := $(shell command -v bash 2>/dev/null || echo /bin/sh)
.SHELLFLAGS := -ec

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Git Revision: $(GIT_REV)"
	@echo "Container Tool: $(CONTAINER_TOOL)"
	@echo "Target Image: $(IMG)"

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt $(GO_BUILD_FLAGS) ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet $(GO_BUILD_FLAGS) ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Fix linting issues automatically.
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: fmt vet ## Run unit tests
	go test $(GO_TEST_FLAGS) ./... -coverprofile cover.out

.PHONY: deps-update
deps-update: ## Update OADP dependencies (set BRANCH=oadp-dev or BRANCH=oadp-X.Y if needed)
	@echo "🔄 Updating dependencies..."
	@BRANCH="$(or $(BRANCH),$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null))"; \
	echo "📦 Using branch: $$BRANCH"; \
	if [ -n "$$BRANCH" ] && echo "$$BRANCH" | grep -qE '^(oadp-dev|oadp-[0-9]+\.[0-9]+)$$'; then \
		echo "✅ Valid OADP branch detected, updating with branch suffix..."; \
		go get github.com/openshift/oadp-operator@$$BRANCH && \
		if echo "$$BRANCH" | grep -qE '^oadp-1\.[34]$$'; then \
			(go get github.com/migtools/oadp-non-admin@$$BRANCH 2>/dev/null || \
			 echo "⚠️  github.com/migtools/oadp-non-admin@$$BRANCH not found, skipping (not available in $$BRANCH)"); \
		else \
			go get github.com/migtools/oadp-non-admin@$$BRANCH; \
		fi; \
	elif [ -n "$$BRANCH" ]; then \
		echo "⚠️  Branch '$$BRANCH' is not a standard OADP branch (oadp-dev, oadp-X.Y)"; \
		echo "💡 Use: make deps-update BRANCH=oadp-dev or make deps-update BRANCH=oadp-X.Y"; \
		echo "📦 Updating dependencies without branch suffix..."; \
	else \
		echo "📦 No branch detected, updating dependencies without branch suffix..."; \
	fi && go mod tidy
	go mod verify
	@echo "✅ Dependencies updated!"

##@ Build

.PHONY: build
build: fmt vet ## Build the binary
	@if [ -n "$(PLATFORM)" ]; then \
		echo "Building $(BINARY_NAME) for $(PLATFORM)..."; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o $(BINARY_NAME)-$(GOOS)-$(GOARCH) $(GO_BUILD_FLAGS) cmd/main.go; \
		echo "✅ Built $(BINARY_NAME)-$(GOOS)-$(GOARCH) successfully!"; \
	else \
		echo "Building $(BINARY_NAME) for current platform ($$(go env GOOS)/$$(go env GOARCH))..."; \
		go build -o $(BINARY_NAME) $(GO_BUILD_FLAGS) cmd/main.go; \
		echo "✅ Built $(BINARY_NAME) successfully!"; \
	fi

build-all: ## Build binaries for all supported platforms
	@echo "🔨 Building for all platforms..."
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		OUTFILE="./$(BINARY_NAME)-$$GOOS-$$GOARCH"; \
		echo "➡️  Building for $$GOOS/$$GOARCH -> $$OUTFILE..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build -mod=mod -o "$$OUTFILE" ./cmd/main.go || { echo "❌ Failed to build for $$platform"; exit 1; }; \
		echo "✅ Built $$OUTFILE"; \
	done
	@echo "✅ All builds completed successfully."

.PHONY: run
run: build ## Build and run the must-gather tool locally
	@BINARY="$(BINARY_NAME)"; \
	if [ -n "$(PLATFORM)" ]; then \
		BINARY="$(BINARY_NAME)-$(GOOS)-$(GOARCH)"; \
	fi; \
	if [ ! -x "$$BINARY" ]; then \
		echo "❌ Binary $$BINARY not found. Run 'make build' first."; \
		exit 1; \
	fi; \
	echo "Running $$BINARY..."; \
	./$$BINARY $(ARGS)

##@ Container

.PHONY: container-build
container-build: ## Build container image
	@echo "Building container image: $(IMG)"
	$(CONTAINER_TOOL) build -t $(IMG) -f $(CONTAINERFILE) . $(CONTAINER_BUILD_ARGS)
	@echo "✅ Container image built: $(IMG)"

.PHONY: container-push
container-push: ## Push container image
	@echo "Pushing container image: $(IMG)"
	$(CONTAINER_TOOL) push $(IMG)
	@echo "✅ Container image pushed: $(IMG)"

.PHONY: container-build-push
container-build-push: container-build container-push ## Build and push container image

##@ Testing

.PHONY: test-must-gather
test-must-gather: build ## Test must-gather locally (requires valid kubeconfig)
	@echo "Testing must-gather binary..."
	./$(BINARY_NAME) --help
	@echo "✅ Must-gather help command works!"

.PHONY: test-container
test-container: container-build ## Test must-gather in container with oc adm must-gather
	@if ! command -v oc >/dev/null 2>&1; then \
		echo "❌ oc command not found. Please install OpenShift CLI."; \
		exit 1; \
	fi
	@echo "Testing container with oc adm must-gather..."
	@echo "Image: $(IMG)"
	@echo "Running: oc adm must-gather --image=$(IMG) -- /usr/bin/gather --help"
	oc adm must-gather --image=$(IMG) -- /usr/bin/gather --help
	@echo "✅ Container test completed!"

##@ Cleanup

.PHONY: clean
clean: ## Remove built binaries and test artifacts
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*-*
	@rm -f cover.out
	@rm -f omg.Dockerfile
	@echo "✅ Cleanup complete!"

.PHONY: container-clean
container-clean: ## Remove container images
	@echo "Removing container images..."
	-$(CONTAINER_TOOL) rmi $(IMG) 2>/dev/null || true
	-$(CONTAINER_TOOL) rmi omg-container 2>/dev/null || true
	@echo "✅ Container cleanup complete!"

##@ Tools

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.61.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	@if [ -f $(GOLANGCI_LINT) ] && $(GOLANGCI_LINT) --version | grep -q $(GOLANGCI_LINT_VERSION); then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) is already installed"; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/main/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION); \
	fi

