# API Proto Makefile
# Build automation for Protocol Buffer definitions

.PHONY: all lint generate breaking clean test help install-tools

# Default target
all: lint generate

# Install required tools
install-tools:
	@echo "Installing Buf CLI..."
	@which buf > /dev/null || (echo "Please install buf: https://buf.build/docs/installation" && exit 1)
	@echo "Buf CLI is installed"

# Lint proto files
lint:
	@echo "Running buf lint..."
	buf lint

# Generate code for all languages
generate:
	@echo "Running buf generate..."
	buf generate
	@echo "Code generation complete"

# Check for breaking changes against main branch
breaking:
	@echo "Checking for breaking changes..."
	buf breaking --against '.git#branch=main'

# Check for breaking changes against a specific tag
breaking-tag:
	@echo "Checking for breaking changes against $(TAG)..."
	buf breaking --against '.git#tag=$(TAG)'

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf gen/
	rm -rf openapi/v1/*.json
	rm -rf openapi/v1/*.yaml
	@echo "Clean complete"

# Run tests (Python property-based tests)
test:
	@echo "Running property-based tests..."
	cd tests && python -m pytest -v --tb=short

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd tests && python -m pytest -v --cov=. --cov-report=html --cov-report=term

# Update buf.lock dependencies
deps:
	@echo "Updating dependencies..."
	buf dep update

# Format proto files (if buf format is available)
format:
	@echo "Formatting proto files..."
	buf format -w

# Validate buf configuration
validate:
	@echo "Validating buf configuration..."
	buf config ls-lint-rules
	buf config ls-breaking-rules

# Generate Go code only
generate-go:
	@echo "Generating Go code..."
	buf generate --template buf.gen.yaml --include-imports --path proto

# Generate TypeScript code only
generate-ts:
	@echo "Generating TypeScript code..."
	buf generate --template buf.gen.yaml --include-imports --path proto

# Generate OpenAPI specs only
generate-openapi:
	@echo "Generating OpenAPI specs..."
	buf generate --template buf.gen.yaml --include-imports --path proto

# Build all and run tests
ci: lint breaking generate test
	@echo "CI pipeline complete"

# Help target
help:
	@echo "Available targets:"
	@echo "  all            - Run lint and generate (default)"
	@echo "  install-tools  - Check/install required tools"
	@echo "  lint           - Lint proto files with buf"
	@echo "  generate       - Generate code for all languages"
	@echo "  breaking       - Check for breaking changes vs main"
	@echo "  breaking-tag   - Check breaking changes vs TAG=<tag>"
	@echo "  clean          - Remove generated files"
	@echo "  test           - Run property-based tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  deps           - Update buf.lock dependencies"
	@echo "  format         - Format proto files"
	@echo "  validate       - Validate buf configuration"
	@echo "  generate-go    - Generate Go code only"
	@echo "  generate-ts    - Generate TypeScript code only"
	@echo "  generate-openapi - Generate OpenAPI specs only"
	@echo "  ci             - Run full CI pipeline"
	@echo "  help           - Show this help message"
