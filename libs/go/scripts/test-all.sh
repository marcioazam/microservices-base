#!/bin/bash
# Unified test runner for all Go library packages

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Running all tests..."

cd "$ROOT_DIR/src"

# Run unit tests
echo "=== Unit Tests ==="
go test ./... -v -count=1

# Run property tests with 100 iterations
echo "=== Property Tests ==="
go test ./... -v -count=1 -run "Property"

# Run benchmarks
echo "=== Benchmarks ==="
go test ./... -bench=. -benchmem -run=^$

echo "All tests completed successfully!"
