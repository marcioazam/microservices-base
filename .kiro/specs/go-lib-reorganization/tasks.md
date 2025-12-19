# Implementation Plan: Go Library Reorganization

## Overview

This plan reorganizes ~45 packages in `libs/go` into 11 domain categories. Tasks are ordered to minimize risk: create structure first, move packages, update references, then validate.

## Tasks

- [x] 1. Create category directory structure
  - Create all category directories: collections, concurrency, functional, optics, patterns, events, server, utils, testing
  - Note: resilience/ and grpc/ already exist with correct structure
  - _Requirements: 1.1_

- [x] 2. Move collections packages
  - [x] 2.1 Move maps, set, slices, queue, pqueue, lru, sort to collections/
    - Move each package directory preserving all files (source, tests, go.mod, go.sum, README)
    - Update go.mod module path to `github.com/auth-platform/libs/go/collections/{package}`
    - _Requirements: 1.2, 5.1, 7.1_

  - [x] 2.2 Write property test for collections structure
    - **Property 1: Category Structure Completeness** (partial - collections)
    - **Validates: Requirements 1.2**

- [x] 3. Move concurrency packages
  - [x] 3.1 Move async, atomic, channels, errgroup, once, pool, syncmap, waitgroup to concurrency/
    - Move each package directory preserving all files
    - Update go.mod module path to `github.com/auth-platform/libs/go/concurrency/{package}`
    - _Requirements: 1.3, 5.1, 7.1_

  - [x] 3.2 Write property test for concurrency structure
    - **Property 1: Category Structure Completeness** (partial - concurrency)
    - **Validates: Requirements 1.3**

- [x] 4. Move functional packages
  - [x] 4.1 Move either, iterator, lazy, option, pipeline, result, stream, tuple to functional/
    - Move each package directory preserving all files
    - Update go.mod module path to `github.com/auth-platform/libs/go/functional/{package}`
    - _Requirements: 1.4, 5.1, 7.1_

  - [x] 4.2 Write property test for functional structure
    - **Property 1: Category Structure Completeness** (partial - functional)
    - **Validates: Requirements 1.4**

- [x] 5. Move optics packages
  - [x] 5.1 Move lens, prism to optics/
    - Move each package directory preserving all files
    - Update go.mod module path to `github.com/auth-platform/libs/go/optics/{package}`
    - _Requirements: 5.1, 7.1_

- [x] 6. Move patterns packages
  - [x] 6.1 Move registry, spec to patterns/
    - Move each package directory preserving all files
    - Update go.mod module path to `github.com/auth-platform/libs/go/patterns/{package}`
    - _Requirements: 5.1, 7.1_

- [x] 7. Move events packages
  - [x] 7.1 Move eventbus, pubsub to events/, rename events/ to events/builder/
    - First rename existing events/ to events/builder/
    - Move eventbus/ and pubsub/ to events/
    - Update go.mod module paths
    - _Requirements: 5.1, 7.1_

- [x] 8. Move server packages
  - [x] 8.1 Move health, tracing to server/, keep server/shutdown/
    - Move health/ to server/health/
    - Move tracing/ to server/tracing/
    - server/shutdown/ already in correct location
    - Update go.mod module paths
    - _Requirements: 5.1, 7.1_

- [x] 9. Move utils packages
  - [x] 9.1 Move audit, cache, codec, diff, error, merge, uuid, validated, validator to utils/
    - Move each package directory preserving all files
    - Update go.mod module path to `github.com/auth-platform/libs/go/utils/{package}`
    - _Requirements: 5.1, 7.1_

- [x] 10. Move testing packages
  - [x] 10.1 Move testutil to testing/
    - Move testutil/ to testing/testutil/
    - Update go.mod module path
    - _Requirements: 5.1, 7.1_

- [x] 11. Checkpoint - Verify package moves
  - Ensure all packages moved to correct locations
  - Verify no orphaned files in old locations
  - Ask the user if questions arise

- [x] 12. Update go.work file
  - [x] 12.1 Regenerate libs/go/go.work with all new module paths
    - List all modules in new category structure
    - Remove old flat structure paths
    - _Requirements: 2.1, 2.2_

  - [x] 12.2 Write property test for go.work completeness
    - **Property 2: Go.work Module Completeness**
    - **Validates: Requirements 2.2**

- [x] 13. Create category README files
  - [x] 13.1 Create README.md for each category directory
    - collections/README.md - Data structures overview
    - concurrency/README.md - Concurrency primitives overview
    - functional/README.md - Functional types overview
    - optics/README.md - Functional optics overview
    - patterns/README.md - Design patterns overview
    - events/README.md - Event handling overview
    - server/README.md - Server utilities overview
    - utils/README.md - General utilities overview
    - testing/README.md - Test utilities overview
    - _Requirements: 3.2_

  - [x] 13.2 Write property test for category README existence
    - **Property 6: Category Documentation Existence**
    - **Validates: Requirements 3.2**

- [x] 14. Update root README.md
  - [x] 14.1 Update libs/go/README.md with new category-based index
    - List all categories with descriptions
    - Include migration guide section
    - Document old → new import path mappings
    - _Requirements: 3.1, 6.1, 6.2_

- [x] 15. Checkpoint - Verify documentation
  - Ensure all READMEs created
  - Verify migration guide is complete
  - Ask the user if questions arise

- [x] 16. Update resilience-service imports
  - [x] 16.1 Update platform/resilience-service import statements
    - Find all imports of libs/go packages
    - Update to new category-based paths
    - Update go.mod replace directives
    - _Requirements: 5.3_

- [x] 17. Build validation
  - [x] 17.1 Run go build ./... in libs/go
    - Verify all packages build successfully
    - Fix any import path issues
    - _Requirements: 8.1_

  - [x] 17.2 Run go build ./... in platform/resilience-service
    - Verify consumer builds successfully
    - Fix any remaining import issues
    - _Requirements: 8.3_

- [x] 18. Test validation
  - [x] 18.1 Run go test ./... in libs/go
    - Verify all tests pass
    - Fix any test failures
    - _Requirements: 4.2, 8.2_

  - [x] 18.2 Write property test for test colocation
    - **Property 3: Test File Colocation**
    - **Validates: Requirements 4.1**

  - [x] 18.3 Write property test for test package naming
    - **Property 4: Test Package Naming Consistency**
    - **Validates: Requirements 4.3**

  - [x] 18.4 Write property test for module path consistency
    - **Property 5: Module Path Directory Consistency**
    - **Validates: Requirements 5.1, 7.1, 7.3**

- [x] 19. Final checkpoint - Full validation
  - Run all property tests
  - Run go build ./... in libs/go
  - Run go test ./... in libs/go
  - Run go build ./... in platform/resilience-service
  - Ensure all tests pass, ask the user if questions arise

## Notes

- All tasks are required including property-based tests
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- The resilience/ and grpc/ directories already have correct structure and don't need reorganization
