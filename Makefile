BINARY     := bctl
MODULE     := github.com/smichalabs/britivectl
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -ldflags "-X $(MODULE)/pkg/version.Version=$(VERSION) \
                         -X $(MODULE)/pkg/version.Commit=$(COMMIT) \
                         -X $(MODULE)/pkg/version.BuildDate=$(BUILD_DATE)"

.PHONY: build test lint clean install snapshot release-dry completions bootstrap help

bootstrap: ## Install all dev tools and git hooks (run once after clone)
	@echo "==> Installing brew tools..."
	brew install pre-commit gitleaks gosec goreleaser 2>/dev/null || true
	@echo "==> Installing Go tools..."
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@cp $(shell go env GOPATH)/bin/golangci-lint $(shell brew --prefix)/bin/golangci-lint
	@echo "==> Installing git hooks..."
	pre-commit install --install-hooks
	@echo "==> Bootstrap complete. Run 'make build' to verify."

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build bin/bctl with version info injected
	go build $(LDFLAGS) -o bin/$(BINARY) .

TEST_PKGS := $(shell find . -name '*_test.go' | xargs -I{} dirname {} | sort -u | sed 's|^\./||' | sed 's|^|$(MODULE)/|')

test: ## Run tests with race detector and coverage
	go test -v -race -cover $(TEST_PKGS)

lint: ## Run golangci-lint
	golangci-lint run ./...

clean: ## Remove build artifacts (bin/, dist/, coverage.out)
	rm -rf bin/ dist/ coverage.out

INSTALL_DIR ?= /opt/homebrew/bin

install: build ## Build and install to $(INSTALL_DIR)
	cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)

snapshot: ## Build release binaries locally via goreleaser (no publish)
	goreleaser release --snapshot --clean

release-dry: ## Full release dry-run via goreleaser (no publish)
	goreleaser release --skip=publish --clean

completions: build ## Generate bash/zsh/fish shell completions
	mkdir -p completions
	./bin/$(BINARY) completion bash > completions/$(BINARY).bash
	./bin/$(BINARY) completion zsh  > completions/$(BINARY).zsh
	./bin/$(BINARY) completion fish > completions/$(BINARY).fish
