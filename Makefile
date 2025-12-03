MAIN_FILE=cmd/crawler/main.go

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
.PHONY: lint
lint: assert_golangci_lint_installed ## Run code linters.
	golangci-lint run ./... --concurrency 2 -v -c .golangci.yaml

.PHONY: vet
vet: assert_go_installed ## Run go vet.
	go vet ./...

.PHONY: test
test: assert_go_installed ## Run tests.
	go test -v ./...

.PHONY: vuln
vuln: assert_govulncheck_installed ## Run govulncheck.
	govulncheck ./...

.PHONY: ci
ci: fumpt generate lint vet test vuln ## Run aci ll checks.

.PHONY: generate
generate: assert_go_installed ## Run code generation.
	go generate ./...

.PHONY: fumpt
fumpt: assert_gofumpt_installed ## Run gofumpt to fix all formatting issues in project.
	gofumpt -w .

.PHONY: build
build: assert_go_installed ## Build binary.
	GOOS=darwin GOARCH=arm64 go build -o bolan.macos.arm64 $(MAIN_FILE)
	chmod +x bolan.macos.arm64
	GOOS=darwin GOARCH=amd64 go build -o bolan.macos.amd64 $(MAIN_FILE)
	chmod +x bolan.macos.amd64

.PHONY: mod
mod: assert_go_installed ## Update go modules.
	go get -u -t ./...
	go mod tidy
	go mod vendor

.PHONY: install
install: assert_go_installed ## Install binary locally.
	go install $(MAIN_FILE)


##@ Assertions
.PHONY: assert_govulncheck_installed
assert_govulncheck_installed: ## Assert govulncheck is installed.
	@if ! command -v govulncheck &> /dev/null; then \
		echo "go vulncheck is not installed; you need to install it in order to run this command"; \
		exit 1; \
	fi
.PHONY: assert_go_installed
assert_go_installed: ## Assert go is installed.
	@if ! command -v go &> /dev/null; then \
		echo "go is not installed; you need to install it in order to run this command"; \
		exit 1; \
	fi

.PHONY: assert_golangci_lint_installed
assert_golangci_lint_installed: ## Assert golangci-lint is installed.
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint is not installed; you need to install it in order to run this command"; \
		exit 1; \
	fi

.PHONY: assert_gofumpt_installed
assert_gofumpt_installed: ## Assert gofumpt is installed.
	@if ! command -v gofumpt &> /dev/null; then \
		echo "gofumpt is not installed; you need to install it in order to run this command"; \
		exit 1; \
	fi
