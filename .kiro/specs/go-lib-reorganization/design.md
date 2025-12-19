# Design Document: Go Library Reorganization

## Overview

This design describes the reorganization of ~45 packages in `libs/go` into logical domain categories. The current flat structure makes it difficult to discover related packages. The new structure groups packages by functionality while maintaining Go idioms (tests alongside source, proper module paths).

## Architecture

### Current State

```
libs/go/
├── async/          ├── lazy/           ├── queue/
├── atomic/         ├── lens/           ├── registry/
├── audit/          ├── lru/            ├── resilience/
├── cache/          ├── maps/           ├── result/
├── channels/       ├── merge/          ├── server/
├── codec/          ├── once/           ├── set/
├── diff/           ├── option/         ├── slices/
├── either/         ├── pipeline/       ├── sort/
├── errgroup/       ├── pool/           ├── spec/
├── error/          ├── pqueue/         ├── stream/
├── eventbus/       ├── prism/          ├── syncmap/
├── events/         ├── pubsub/         ├── testutil/
├── grpc/                               ├── tracing/
├── health/                             ├── tuple/
├── iterator/                           ├── uuid/
                                        ├── validated/
                                        ├── validator/
                                        ├── waitgroup/
```

### Target State

```
libs/go/
├── collections/           # Data structures
│   ├── maps/
│   ├── set/
│   ├── slices/
│   ├── queue/
│   ├── pqueue/
│   ├── lru/
│   └── sort/
├── concurrency/           # Concurrency primitives
│   ├── async/
│   ├── atomic/
│   ├── channels/
│   ├── errgroup/
│   ├── once/
│   ├── pool/
│   ├── syncmap/
│   └── waitgroup/
├── functional/            # Functional programming types
│   ├── either/
│   ├── iterator/
│   ├── lazy/
│   ├── option/
│   ├── pipeline/
│   ├── result/
│   ├── stream/
│   └── tuple/
├── optics/                # Functional optics (lens, prism)
│   ├── lens/
│   └── prism/
├── patterns/              # Design patterns
│   ├── registry/
│   └── spec/
├── events/                # Event handling
│   ├── builder/           # (from events/)
│   ├── eventbus/
│   └── pubsub/
├── resilience/            # Fault tolerance (existing structure)
│   ├── bulkhead/
│   ├── circuitbreaker/
│   ├── domain/
│   ├── errors/
│   ├── ratelimit/
│   ├── retry/
│   └── timeout/
├── server/                # Server utilities
│   ├── health/
│   ├── shutdown/          # (from server/shutdown/)
│   └── tracing/
├── grpc/                  # gRPC utilities
│   └── errors/
├── utils/                 # General utilities
│   ├── audit/
│   ├── cache/
│   ├── codec/
│   ├── diff/
│   ├── error/
│   ├── merge/
│   ├── uuid/
│   ├── validated/
│   └── validator/
└── testing/               # Test utilities
    └── testutil/
```

## Components and Interfaces

### Package Migration Map

| Current Location | New Location | Category |
|-----------------|--------------|----------|
| `async/` | `concurrency/async/` | Concurrency |
| `atomic/` | `concurrency/atomic/` | Concurrency |
| `audit/` | `utils/audit/` | Utils |
| `cache/` | `utils/cache/` | Utils |
| `channels/` | `concurrency/channels/` | Concurrency |
| `codec/` | `utils/codec/` | Utils |
| `diff/` | `utils/diff/` | Utils |
| `either/` | `functional/either/` | Functional |
| `errgroup/` | `concurrency/errgroup/` | Concurrency |
| `error/` | `utils/error/` | Utils |
| `eventbus/` | `events/eventbus/` | Events |
| `events/` | `events/builder/` | Events |
| `grpc/` | `grpc/` | gRPC (unchanged) |
| `health/` | `server/health/` | Server |
| `iterator/` | `functional/iterator/` | Functional |
| `lazy/` | `functional/lazy/` | Functional |
| `lens/` | `optics/lens/` | Optics |
| `lru/` | `collections/lru/` | Collections |
| `maps/` | `collections/maps/` | Collections |
| `merge/` | `utils/merge/` | Utils |
| `once/` | `concurrency/once/` | Concurrency |
| `option/` | `functional/option/` | Functional |
| `pipeline/` | `functional/pipeline/` | Functional |
| `pool/` | `concurrency/pool/` | Concurrency |
| `pqueue/` | `collections/pqueue/` | Collections |
| `prism/` | `optics/prism/` | Optics |
| `pubsub/` | `events/pubsub/` | Events |
| `queue/` | `collections/queue/` | Collections |
| `registry/` | `patterns/registry/` | Patterns |
| `resilience/` | `resilience/` | Resilience (unchanged) |
| `result/` | `functional/result/` | Functional |
| `server/shutdown/` | `server/shutdown/` | Server |
| `set/` | `collections/set/` | Collections |
| `slices/` | `collections/slices/` | Collections |
| `sort/` | `collections/sort/` | Collections |
| `spec/` | `patterns/spec/` | Patterns |
| `stream/` | `functional/stream/` | Functional |
| `syncmap/` | `concurrency/syncmap/` | Concurrency |
| `testutil/` | `testing/testutil/` | Testing |
| `tracing/` | `server/tracing/` | Server |
| `tuple/` | `functional/tuple/` | Functional |
| `uuid/` | `utils/uuid/` | Utils |
| `validated/` | `utils/validated/` | Utils |
| `validator/` | `utils/validator/` | Utils |
| `waitgroup/` | `concurrency/waitgroup/` | Concurrency |

### Module Path Convention

Each package module path follows the pattern:
```
github.com/auth-platform/libs/go/{category}/{package}
```

Example:
- Old: `github.com/auth-platform/libs/go/slices`
- New: `github.com/auth-platform/libs/go/collections/slices`

### Go Workspace Configuration

```go
// libs/go/go.work
go 1.21

use (
    ./collections/maps
    ./collections/set
    ./collections/slices
    ./collections/queue
    ./collections/pqueue
    ./collections/lru
    ./collections/sort
    ./concurrency/async
    ./concurrency/atomic
    ./concurrency/channels
    ./concurrency/errgroup
    ./concurrency/once
    ./concurrency/pool
    ./concurrency/syncmap
    ./concurrency/waitgroup
    ./functional/either
    ./functional/iterator
    ./functional/lazy
    ./functional/option
    ./functional/pipeline
    ./functional/result
    ./functional/stream
    ./functional/tuple
    ./optics/lens
    ./optics/prism
    ./patterns/registry
    ./patterns/spec
    ./events/builder
    ./events/eventbus
    ./events/pubsub
    ./resilience/bulkhead
    ./resilience/circuitbreaker
    ./resilience/domain
    ./resilience/errors
    ./resilience/ratelimit
    ./resilience/retry
    ./resilience/timeout
    ./server/health
    ./server/shutdown
    ./server/tracing
    ./grpc/errors
    ./utils/audit
    ./utils/cache
    ./utils/codec
    ./utils/diff
    ./utils/error
    ./utils/merge
    ./utils/uuid
    ./utils/validated
    ./utils/validator
    ./testing/testutil
)
```

## Data Models

### Directory Structure Model

Each category directory contains:
1. `README.md` - Category overview and package list
2. Package subdirectories with:
   - `*.go` - Source files
   - `*_test.go` - Test files (same directory)
   - `go.mod` - Module definition
   - `go.sum` - Dependency checksums (if dependencies exist)
   - `README.md` - Package documentation (optional)

### Import Path Update Model

For consumers (e.g., `platform/resilience-service`):

```go
// Before
import "github.com/auth-platform/libs/go/slices"

// After
import "github.com/auth-platform/libs/go/collections/slices"
```

Replace directives in consumer `go.mod`:
```go
// Before
replace github.com/auth-platform/libs/go/slices => ../../libs/go/slices

// After
replace github.com/auth-platform/libs/go/collections/slices => ../../libs/go/collections/slices
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Category Structure Completeness

*For any* package in the migration map, the package directory SHALL exist in its designated category location with all original source files and test files present.

**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: Go.work Module Completeness

*For any* package directory containing a go.mod file, that module path SHALL be listed in the libs/go/go.work file.

**Validates: Requirements 2.2**

### Property 3: Test File Colocation

*For any* package directory containing source files (*.go excluding *_test.go), all associated test files (*_test.go) SHALL reside in the same directory.

**Validates: Requirements 4.1**

### Property 4: Test Package Naming Consistency

*For any* test file (*_test.go) in a package, the package declaration SHALL match the package name of the source files in the same directory (not using _test suffix for internal tests).

**Validates: Requirements 4.3**

### Property 5: Module Path Directory Consistency

*For any* package with a go.mod file, the module path declared in go.mod SHALL match the package's directory path relative to the repository root (following the pattern `github.com/auth-platform/libs/go/{category}/{package}`).

**Validates: Requirements 5.1, 7.1, 7.3**

### Property 6: Category Documentation Existence

*For any* category directory (collections, concurrency, functional, optics, patterns, events, resilience, server, grpc, utils, testing), a README.md file SHALL exist.

**Validates: Requirements 3.2**

## Error Handling

### Migration Errors

| Error Scenario | Handling Strategy |
|---------------|-------------------|
| Package move fails | Rollback to original location, log error |
| go.mod update fails | Restore original go.mod, report conflict |
| Import path not found | Document in migration guide, provide sed command |
| Circular dependency detected | Analyze and refactor before migration |
| Build fails after migration | Check import paths, verify replace directives |

### Validation Errors

| Error Scenario | Handling Strategy |
|---------------|-------------------|
| go build fails | Report specific package and error, suggest fix |
| go test fails | Report test name and failure, preserve test output |
| Missing go.mod | Create go.mod with correct module path |
| Missing README | Generate template README for category |

## Testing Strategy

### Dual Testing Approach

This reorganization requires both unit tests (specific examples) and property-based tests (universal properties).

### Unit Tests

Unit tests verify specific examples and edge cases:

1. **Build Validation Tests**
   - `go build ./...` succeeds in libs/go
   - `go build ./...` succeeds in platform/resilience-service
   - `go test ./...` passes in libs/go

2. **Structure Validation Tests**
   - go.work file exists at libs/go/go.work
   - Root README.md exists with category index
   - Specific packages exist in expected locations (spot checks)

### Property-Based Tests

Property-based tests verify universal properties across all packages. Each property test runs minimum 100 iterations.

**Testing Framework**: Go's built-in testing with `testing/quick` or custom property generators.

**Property Test Implementation**:

```go
// Example property test structure
func TestProperty_ModulePathConsistency(t *testing.T) {
    // Feature: go-lib-reorganization, Property 5: Module Path Directory Consistency
    packages := discoverAllPackages("libs/go")
    for _, pkg := range packages {
        goModPath := filepath.Join(pkg, "go.mod")
        if !fileExists(goModPath) {
            continue
        }
        modulePath := parseModulePath(goModPath)
        expectedPath := deriveExpectedModulePath(pkg)
        if modulePath != expectedPath {
            t.Errorf("Package %s: module path %s != expected %s", 
                pkg, modulePath, expectedPath)
        }
    }
}
```

### Test Coverage Matrix

| Property | Test Type | Validates |
|----------|-----------|-----------|
| Category Structure | Property | Req 1.1-1.4 |
| Go.work Completeness | Property | Req 2.2 |
| Test Colocation | Property | Req 4.1 |
| Package Naming | Property | Req 4.3 |
| Module Path Consistency | Property | Req 5.1, 7.1, 7.3 |
| Category README | Property | Req 3.2 |
| Build Success | Unit/Example | Req 2.3, 8.1, 8.3 |
| Test Success | Unit/Example | Req 4.2, 8.2 |

### Validation Script

A validation script (`scripts/validate-reorganization.sh`) will:
1. Run all property checks
2. Execute `go build ./...`
3. Execute `go test ./...`
4. Report any violations with specific file/line information
