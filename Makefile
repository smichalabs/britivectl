BINARY     := bctl
MODULE     := github.com/smichalabs/britivectl
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -ldflags "-X $(MODULE)/pkg/version.Version=$(VERSION) \
                         -X $(MODULE)/pkg/version.Commit=$(COMMIT) \
                         -X $(MODULE)/pkg/version.BuildDate=$(BUILD_DATE)"

.PHONY: build test lint security tidy clean install uninstall snapshot release-dry completions bootstrap bootstrap-aws docs docs-serve help

TOOL_PHASE ?= pre

define TOOL_TABLE
	@printf "\n  %-20s %-18s %-12s %s\n" "Tool" "Current" "Expected" "$(if $(filter post,$(TOOL_PHASE)),Result,Action)"
	@echo "  ---------------------------------------------------------------"
	@if command -v go >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "go" "$$(go version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" ">=1.21" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "go" "not installed" ">=1.21" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@if command -v pre-commit >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "pre-commit" "$$(pre-commit --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" "latest" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "pre-commit" "not installed" "latest" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@if command -v golangci-lint >/dev/null 2>&1 && golangci-lint --version 2>&1 | grep -q "version 2"; then \
		printf "  %-20s %-18s %-12s %s\n" "golangci-lint" "$$(golangci-lint --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)" "v2+" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "golangci-lint" "$$(golangci-lint --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)" "v2+" "$(if $(filter post,$(TOOL_PHASE)),upgraded,upgrade)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "golangci-lint" "not installed" "v2+" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@if command -v gitleaks >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "gitleaks" "$$(gitleaks version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" "latest" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "gitleaks" "not installed" "latest" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@if command -v gosec >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "gosec" "$$(gosec --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" "latest" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "gosec" "not installed" "latest" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@if command -v goreleaser >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "goreleaser" "$$(goreleaser --version 2>&1 | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)" "latest" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "goreleaser" "not installed" "latest" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@if command -v govulncheck >/dev/null 2>&1; then \
		printf "  %-20s %-18s %-12s %s\n" "govulncheck" "$$(govulncheck --version 2>&1 | grep Scanner | grep -oE '[0-9]+\.[0-9]+\.[0-9]+')" "latest" "$(if $(filter post,$(TOOL_PHASE)),skipped,skip)"; \
	else \
		printf "  %-20s %-18s %-12s %s\n" "govulncheck" "not installed" "latest" "$(if $(filter post,$(TOOL_PHASE)),installed,install)"; \
	fi
	@echo "  ---------------------------------------------------------------"
endef

bootstrap: ## Install all dev tools and git hooks (run once after clone)
	@echo ""
	@echo "==> Preflight: tool status"
	$(eval TOOL_PHASE := pre)
	$(TOOL_TABLE)
	@echo ""
	@echo "==> Detecting OS..."
	@OS=$$(uname -s); \
	IS_WSL=false; \
	if [ -f /proc/version ] && grep -qi microsoft /proc/version 2>/dev/null; then IS_WSL=true; fi; \
	if [ "$$OS" = "Darwin" ]; then \
		echo "  macOS detected — using Homebrew"; \
		command -v brew >/dev/null 2>&1 || { echo "ERROR: Homebrew not found. Install from https://brew.sh"; exit 1; }; \
		for tool in pre-commit gitleaks gosec goreleaser; do \
			if command -v $$tool >/dev/null 2>&1; then \
				echo "  ✓ $$tool already installed"; \
			else \
				echo "  → installing $$tool..."; \
				brew install $$tool; \
			fi; \
		done; \
	elif [ "$$OS" = "Linux" ]; then \
		if [ "$$IS_WSL" = "true" ]; then \
			echo "  WSL detected"; \
			echo ""; \
			echo "  Checking WSL prerequisites..."; \
			MISSING=""; \
			command -v pip3      >/dev/null 2>&1 || MISSING="$$MISSING python3-pip"; \
			command -v xdg-open  >/dev/null 2>&1 || command -v wslview >/dev/null 2>&1 || MISSING="$$MISSING wslu"; \
			command -v curl      >/dev/null 2>&1 || MISSING="$$MISSING curl"; \
			if [ -n "$$MISSING" ]; then \
				echo ""; \
				echo "  ERROR: Missing required packages. Install them first:"; \
				echo ""; \
				echo "    sudo apt update && sudo apt install -y$$MISSING"; \
				echo ""; \
				exit 1; \
			fi; \
			echo "  ✓ WSL prerequisites satisfied"; \
			echo "  Note: 'bctl login' opens the browser via wslview (wslu)."; \
			echo ""; \
		else \
			echo "  Linux detected"; \
		fi; \
		command -v pre-commit >/dev/null 2>&1 && echo "  ✓ pre-commit already installed" || pip3 install --user pre-commit; \
		command -v gitleaks   >/dev/null 2>&1 && echo "  ✓ gitleaks already installed"   || { \
			echo "  → installing gitleaks..."; \
			ARCH=$$(uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/'); \
			curl -sSL "https://github.com/gitleaks/gitleaks/releases/latest/download/gitleaks_linux_$${ARCH}.tar.gz" \
			| sudo tar -xz -C /usr/local/bin gitleaks; }; \
		command -v gosec      >/dev/null 2>&1 && echo "  ✓ gosec already installed"      || go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		command -v goreleaser >/dev/null 2>&1 && echo "  ✓ goreleaser already installed" || { \
			echo "  → installing goreleaser..."; \
			ARCH=$$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/'); \
			curl -sSL "https://github.com/goreleaser/goreleaser/releases/latest/download/goreleaser_Linux_$${ARCH}.tar.gz" \
			| sudo tar -xz -C /usr/local/bin goreleaser; }; \
	else \
		echo "ERROR: Unsupported OS: $$OS"; exit 1; \
	fi
	@echo "==> Installing golangci-lint v2..."
	@if command -v golangci-lint >/dev/null 2>&1 && golangci-lint --version 2>&1 | grep -q "version 2"; then \
		echo "  ✓ golangci-lint v2 already installed"; \
	else \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
		cp $$(go env GOPATH)/bin/golangci-lint $$(go env GOPATH)/bin/golangci-lint; \
		command -v brew >/dev/null 2>&1 && cp $$(go env GOPATH)/bin/golangci-lint $$(brew --prefix)/bin/golangci-lint || true; \
	fi
	@echo "==> Installing govulncheck..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		echo "  ✓ govulncheck already installed"; \
	else \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		command -v brew >/dev/null 2>&1 && cp $$(go env GOPATH)/bin/govulncheck $$(brew --prefix)/bin/govulncheck || true; \
	fi
	@echo "==> Installing git hooks..."
	pre-commit install --install-hooks
	@echo ""
	@echo "==> Postflight: tool status"
	$(eval TOOL_PHASE := post)
	$(TOOL_TABLE)
	@echo ""
	@echo "==> Bootstrap complete. Run 'make build' to verify your setup."

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build bin/bctl with version info injected
	go build $(LDFLAGS) -o bin/$(BINARY) .

TEST_PKGS := $(shell find . -name '*_test.go' | xargs -I{} dirname {} | sort -u | sed 's|^\./||' | sed 's|^|$(MODULE)/|')

COVERAGE_THRESHOLD ?= 90

test: ## Run tests with race detector and coverage (fails below $(COVERAGE_THRESHOLD)%)
	go test -v -race -coverprofile=coverage.out $(TEST_PKGS)
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,""); print int($$3)}'); \
	echo "Coverage: $${total}% (threshold: $(COVERAGE_THRESHOLD)%)"; \
	if [ "$$total" -lt "$(COVERAGE_THRESHOLD)" ]; then \
		echo "FAIL: coverage $${total}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi

lint: ## Run golangci-lint
	golangci-lint run ./...

security: ## Run security scans (gosec + gitleaks + govulncheck + go mod verify)
	@echo "==> go mod verify: module tamper check"
	go mod verify
	@echo ""
	@echo "==> gosec: Go security analysis"
	gosec -exclude=G104,G204,G302,G304 ./...
	@echo ""
	@echo "==> gitleaks: secret detection"
	gitleaks detect --source . --verbose
	@echo ""
	@echo "==> govulncheck: dependency vulnerability scan"
	govulncheck ./...

tidy: ## Run go mod tidy and verify go.mod/go.sum are clean
	go mod tidy
	go mod verify

clean: ## Remove build artifacts (bin/, dist/, coverage.out)
	rm -rf bin/ dist/ coverage.out

INSTALL_DIR ?= /opt/homebrew/bin

install: build ## Build and install to $(INSTALL_DIR)
	cp bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)

uninstall: ## Remove manually installed bctl from $(INSTALL_DIR) (use before switching to Homebrew)
	rm -f $(INSTALL_DIR)/$(BINARY)

snapshot: ## Build release binaries locally via goreleaser (no publish)
	goreleaser release --snapshot --clean

release-dry: ## Full release dry-run via goreleaser (no publish)
	goreleaser release --skip=publish --clean

setup-secrets: ## Set GitHub Actions secrets from terraform-cli credentials
	./scripts/setup-github-secrets.sh

bootstrap-aws: ## Create least-privilege terraform-cli IAM user (uses root creds — run once)
	./scripts/bootstrap-aws.sh

docs: ## Build docs site locally (output: site/)
	pip3 install -q --break-system-packages mkdocs-material
	python3 -m mkdocs build --strict --site-dir site

docs-serve: ## Serve docs locally with live reload (http://localhost:8000)
	pip3 install -q --break-system-packages mkdocs-material
	python3 -m mkdocs serve

docs-deploy: docs ## Build and deploy docs to S3 + invalidate CloudFront
	AWS_PROFILE=terraform aws s3 sync site/ s3://smichalabs-docs/utils/bctl --delete --cache-control "max-age=300"
	$(eval DIST_ID := $(shell AWS_PROFILE=terraform aws cloudfront list-distributions --query "DistributionList.Items[0].Id" --output text))
	AWS_PROFILE=terraform aws cloudfront create-invalidation --distribution-id $(DIST_ID) --paths "/*"

completions: build ## Generate bash/zsh/fish shell completions
	mkdir -p completions
	./bin/$(BINARY) completion bash > completions/$(BINARY).bash
	./bin/$(BINARY) completion zsh  > completions/$(BINARY).zsh
	./bin/$(BINARY) completion fish > completions/$(BINARY).fish
