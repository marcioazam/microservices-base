.PHONY: build run test test-unit test-integration test-property lint clean docker-build docker-run migrate

# Variables
APP_NAME := file-upload
BUILD_DIR := bin
MAIN_PATH := ./cmd/server

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOLINT := golangci-lint

# Build
build:
	$(GOBUILD) -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# Run
run:
	$(GOCMD) run $(MAIN_PATH)

# Test
test:
	$(GOTEST) -v -race -cover ./...

test-unit:
	$(GOTEST) -v -race -cover ./internal/...

test-integration:
	$(GOTEST) -v -race -tags=integration ./tests/integration/...

test-property:
	$(GOTEST) -v -race -tags=property ./tests/property/...

test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	$(GOLINT) run ./...

# Clean
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker
docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run -p 8080:8080 $(APP_NAME):latest

# Database migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)
