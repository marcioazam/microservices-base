# Auth Platform - Root Makefile
# State-of-the-art 2025 monorepo build system

.PHONY: all build test lint clean docker help

# Default target
all: lint build test

# ============================================
# Build Commands
# ============================================

build: build-rust build-go build-elixir ## Build all services
	@echo "✅ All services built successfully"

build-rust: ## Build Rust services
	@echo "🦀 Building Rust services..."
	cd services/auth-edge && cargo build --release
	cd services/token && cargo build --release

build-go: ## Build Go services
	@echo "🐹 Building Go services..."
	cd services/iam-policy && go build -o bin/iam-policy ./cmd/server

build-elixir: ## Build Elixir services
	@echo "💧 Building Elixir services..."
	cd services/session-identity && mix deps.get && mix compile
	cd services/mfa && mix deps.get && mix compile

# ============================================
# Test Commands
# ============================================

test: test-rust test-go test-elixir ## Run all tests
	@echo "✅ All tests passed"

test-rust: ## Run Rust tests
	@echo "🦀 Testing Rust services..."
	cd services/auth-edge && cargo test
	cd services/token && cargo test
	cd libs/rust/vault && cargo test
	cd libs/rust/caep && cargo test

test-go: ## Run Go tests
	@echo "🐹 Testing Go services..."
	cd services/iam-policy && go test -v -race ./...
	cd libs/go/audit && go test -v ./...
	cd libs/go/tracing && go test -v ./...

test-elixir: ## Run Elixir tests
	@echo "💧 Testing Elixir services..."
	cd services/session-identity && mix test
	cd services/mfa && mix test

test-property: ## Run property-based tests
	@echo "🔬 Running property-based tests..."
	cd services/auth-edge && cargo test --features proptest -- --ignored
	cd services/token && cargo test --features proptest -- --ignored

test-contract: ## Run contract tests
	@echo "📜 Running contract tests..."
	cd services/auth-edge && cargo test --test pact_consumer_tests
	cd services/token && cargo test --test pact_provider_tests

# ============================================
# Lint Commands
# ============================================

lint: lint-rust lint-go lint-elixir ## Lint all code
	@echo "✅ All linting passed"

lint-rust: ## Lint Rust code
	@echo "🦀 Linting Rust..."
	cd services/auth-edge && cargo fmt --check && cargo clippy -- -D warnings
	cd services/token && cargo fmt --check && cargo clippy -- -D warnings

lint-go: ## Lint Go code
	@echo "🐹 Linting Go..."
	cd services/iam-policy && go fmt ./... && go vet ./...

lint-elixir: ## Lint Elixir code
	@echo "💧 Linting Elixir..."
	cd services/session-identity && mix format --check-formatted && mix credo
	cd services/mfa && mix format --check-formatted && mix credo

# ============================================
# Docker Commands
# ============================================

docker-build: ## Build all Docker images
	@echo "🐳 Building Docker images..."
	docker-compose -f deploy/docker/docker-compose.yml build

docker-up: ## Start all services with Docker
	@echo "🐳 Starting services..."
	docker-compose -f deploy/docker/docker-compose.yml up -d

docker-down: ## Stop all services
	@echo "🐳 Stopping services..."
	docker-compose -f deploy/docker/docker-compose.yml down

docker-logs: ## View service logs
	docker-compose -f deploy/docker/docker-compose.yml logs -f

# ============================================
# Proto Commands
# ============================================

proto: ## Generate protobuf code
	@echo "📦 Generating protobuf code..."
	protoc --proto_path=api/proto \
		--go_out=services/iam-policy \
		--go-grpc_out=services/iam-policy \
		api/proto/auth/*.proto api/proto/infra/*.proto

# ============================================
# Clean Commands
# ============================================

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	cd services/auth-edge && cargo clean
	cd services/token && cargo clean
	cd services/iam-policy && rm -rf bin/
	cd services/session-identity && rm -rf _build/
	cd services/mfa && rm -rf _build/

# ============================================
# Help
# ============================================

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
