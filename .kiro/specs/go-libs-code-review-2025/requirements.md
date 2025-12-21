# Requirements: Go Libs Code Review & Redundancy Elimination 2025

## Overview
Comprehensive code review and refactoring of `libs/go/src` to eliminate redundancies, apply Go 2025 best practices, and ensure DRY principles.

## Functional Requirements

### FR-1: Redundancy Detection & Elimination
- FR-1.1: Identify duplicate code patterns across packages
- FR-1.2: Detect similar implementations (LRU in collections vs utils/cache)
- FR-1.3: Find overlapping functionality (uuid in domain vs utils/uuid)
- FR-1.4: Consolidate duplicate error handling patterns
- FR-1.5: Merge similar validation logic

### FR-2: Package Structure Optimization
- FR-2.1: Eliminate nested redundant folders (X/X pattern) - DONE
- FR-2.2: Consolidate related packages (codec exists in root AND utils/codec)
- FR-2.3: Remove empty or stub packages
- FR-2.4: Ensure single responsibility per package

### FR-3: Go 2025 Best Practices
- FR-3.1: Apply modern generics patterns (Go 1.21+)
- FR-3.2: Use type inference where appropriate
- FR-3.3: Implement functional options pattern consistently
- FR-3.4: Apply iterator patterns (Go 1.23 range-over-func)
- FR-3.5: Use structured logging (slog)

### FR-4: Code Quality Standards
- FR-4.1: Max 400 lines per file
- FR-4.2: Cyclomatic complexity ≤ 10
- FR-4.3: Consistent naming conventions
- FR-4.4: Proper error wrapping with context
- FR-4.5: Thread-safety documentation

### FR-5: API Consistency
- FR-5.1: Consistent constructor patterns (New, Of, From)
- FR-5.2: Consistent method naming (Get, Set, Add, Remove)
- FR-5.3: Consistent Option pattern usage
- FR-5.4: Consistent Result/Either usage

## Non-Functional Requirements

### NFR-1: Performance
- Zero-allocation paths where possible
- Efficient generics usage (no boxing)

### NFR-2: Maintainability
- Clear package boundaries
- Minimal inter-package dependencies
- Comprehensive documentation

### NFR-3: Testability
- Property-based tests for core logic
- Test coverage ≥ 80%

## Identified Redundancies (Initial Analysis)

| Location 1 | Location 2 | Type |
|------------|------------|------|
| `collections/lru/` | `collections/lru.go` | Duplicate LRU |
| `collections/queue/` | `collections/queue.go` | Duplicate Queue |
| `collections/set/` | `collections/set.go` | Duplicate Set |
| `collections/pqueue/` | `collections/pqueue.go` | Duplicate PQueue |
| `domain/uuid.go` | `utils/uuid/` | Duplicate UUID |
| `codec/` | `utils/codec/` | Duplicate Codec |
| `functional/option.go` | `functional/option/` | Duplicate Option |
| `functional/either.go` | `functional/either/` | Duplicate Either |
| `functional/result.go` | `functional/result/` | Duplicate Result |
| `server/health.go` | `server/health/` | Duplicate Health |
| `patterns/registry.go` | `patterns/registry/` | Duplicate Registry |
| `patterns/spec.go` | `patterns/spec/` | Duplicate Spec |
| `grpc/errors.go` | `grpc/errors_sub/` | Duplicate Errors |
| `optics/lens.go` | `optics/lens/` | Duplicate Lens |

## Success Criteria
- [ ] Zero duplicate implementations
- [ ] All packages follow single responsibility
- [ ] 100% Go 2025 best practices compliance
- [ ] All files < 400 lines
- [ ] Test coverage ≥ 80%
