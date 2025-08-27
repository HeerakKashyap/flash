# GoFlash Framework Makefile

.PHONY: help test build fmt lint mod-tidy clean install-hooks pre-release

# Default target
help: ## Show this help message
	@echo "GoFlash Framework Development Commands"
	@echo "═══════════════════════════════════════"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Performance Testing:"
	@echo "  cd internal/performance && make help"

# Core Testing
test: ## Run all tests
	go test ./...

test-verbose: ## Run all tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Development
build: ## Build the project
	go build ./...

fmt: ## Format code
	go fmt ./...

lint: ## Run linter (requires golangci-lint)
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, skipping lint check" && exit 0)
	golangci-lint run

mod-tidy: ## Clean up go.mod
	go mod tidy

# Cleanup
clean: ## Clean up generated files
	rm -f coverage.out coverage.html
	rm -f *.test

# Git hooks
install-hooks: ## Install git hooks for performance testing
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/bash' > .git/hooks/pre-commit
	@echo 'echo "Running performance baseline check..."' >> .git/hooks/pre-commit
	@echo 'cd internal/performance && make perf-baseline' >> .git/hooks/pre-commit
	@echo 'if [ $$? -ne 0 ]; then' >> .git/hooks/pre-commit
	@echo '    echo "❌ Performance tests failed"' >> .git/hooks/pre-commit
	@echo '    exit 1' >> .git/hooks/pre-commit
	@echo 'fi' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Pre-commit hook installed (runs performance tests)"

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Release preparation
pre-release: test lint ## Run all checks before release
	@echo "Running performance baseline check..."
	@cd internal/performance && make perf-baseline
	@echo "✅ All pre-release checks passed"