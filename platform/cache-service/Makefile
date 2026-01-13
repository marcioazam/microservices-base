.PHONY: build test lint run clean docker proto

# Variables
BINARY_NAME=cache-service
DOCKER_IMAGE=platform/cache-service
VERSION?=latest

# Build
build:
	go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/cache-service

# Test
test:
	go test -v -race -cover ./...

test-property:
	go test -v -race ./tests/property/...

# Lint
lint:
	golangci-lint run ./...

# Run locally
run:
	go run ./cmd/cache-service

# Clean
clean:
	rm -rf bin/
	go clean

# Docker
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-push:
	docker push $(DOCKER_IMAGE):$(VERSION)

# Proto (requires protoc)
proto:
	protoc --go_out=. --go-grpc_out=. api/proto/cache/v1/cache.proto

# Dependencies
deps:
	go mod download
	go mod tidy

# Format
fmt:
	go fmt ./...
	goimports -w .
