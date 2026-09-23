BINARY_NAME  := terraform-provider-openmetadata
MODULE       := github.com/Codility/terraform-provider-openmetadata
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
LDFLAGS      := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

OS           := $(shell go env GOOS 2>/dev/null || echo linux)
ARCH         := $(shell go env GOARCH 2>/dev/null || echo amd64)

# Terraform plugin directory
TF_PLUGIN_DIR := $(HOME)/.terraform.d/plugins/registry.terraform.io/codility/openmetadata/$(VERSION)/$(OS)_$(ARCH)

.PHONY: build install clean fmt lint test testacc testacc-external update-test-compose docs codegen deps help

## Build the provider binary
build:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .

## Build + install into the Terraform plugin directory
install: build
	mkdir -p $(TF_PLUGIN_DIR)
	cp $(BINARY_NAME) $(TF_PLUGIN_DIR)/$(BINARY_NAME)
	@echo "Installed to $(TF_PLUGIN_DIR)/$(BINARY_NAME)"

## Format Go code
fmt:
	go fmt ./...

## Run linter
lint:
	golangci-lint run ./...

## Run unit tests (no live OpenMetadata required)
test:
	go test -v -count=1 ./...

## Run acceptance tests against a local docker-compose OpenMetadata stack.
## Starts the stack, acquires a JWT, runs TF_ACC=1 tests, and tears down.
testacc:
	@bash scripts/testacc.sh

## Run acceptance tests against an already-running OpenMetadata instance.
## Requires OPENMETADATA_HOST and OPENMETADATA_TOKEN to be exported.
##   export OPENMETADATA_HOST=http://localhost:8585
##   export OPENMETADATA_TOKEN=<jwt-token>
##   make testacc-external
testacc-external:
	TF_ACC=1 go test -v -count=1 -timeout 30m ./internal/provider/...

## Re-download the official OpenMetadata docker-compose.yml for the version
## pinned in docker/test/.env. Run this when bumping OPENMETADATA_VERSION,
## then commit both docker/test/docker-compose.yml and docker/test/.env.
update-test-compose:
	@bash scripts/update-test-compose.sh

## Generate provider documentation (requires tfplugindocs)
docs:
	go generate ./...

## Generate a resource — use the AI codegen skill instead of this target.
## See: .github/skills/codegen/SKILL.md
codegen:
	@echo "Resource generation is now handled by the AI codegen skill."
	@echo "See: .github/skills/codegen/SKILL.md"
	@echo ""
	@echo "Ask your AI agent: 'Generate a TF resource for <entity>'"

## Resolve Go module dependencies
deps:
	go mod tidy

## Remove build artifacts
clean:
	rm -f $(BINARY_NAME)

## Show available targets
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Environment:"
	@echo "  VERSION=$(VERSION)"
	@echo "  OS=$(OS) ARCH=$(ARCH)"
