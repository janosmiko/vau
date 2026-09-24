.PHONY: help setup lint lint-fix test coverage goreleaser-check build

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup: ## Configure git hooks
	git config core.hooksPath .githooks

lint: ## Run golangci-lint
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with autofix
	golangci-lint run --fix ./...

test: ## Run tests
	go test ./...

coverage: ## Run tests with coverage report
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in your browser for details"

goreleaser-check: ## Validate .goreleaser.yaml and run a snapshot release without publishing
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "error: goreleaser is required (https://goreleaser.com/install/)"; exit 1; \
	}
	goreleaser check --config .goreleaser.yaml
	goreleaser release --skip=publish,sign,docker --snapshot --clean --config .goreleaser.yaml
	@echo ""
	@echo "Snapshot artifacts in ./dist/"
	@ls -la dist/

build: setup ## Build the vau binary
	go build -o vau .

.DEFAULT_GOAL := help
