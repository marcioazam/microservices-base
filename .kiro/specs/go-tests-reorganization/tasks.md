# Implementation Plan: Go Tests Reorganization

## Overview

This plan implements the reorganization of Go test files from scattered locations in `libs/go` to the centralized `libs/go/tests` directory using PowerShell commands.

## Tasks

- [x] 1. Discover and list all test files to be moved
  - Execute PowerShell command to find all `*_test.go` files outside `tests/`
  - Generate a complete list of source files with their relative paths
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2. Move collections module tests
  - [x] 2.1 Move `libs/go/collections/lru/lru_test.go` to `libs/go/tests/collections/lru/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 2.2 Move `libs/go/collections/pqueue/pqueue_test.go` to `libs/go/tests/collections/pqueue/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 2.3 Move `libs/go/collections/queue/queue_test.go` to `libs/go/tests/collections/queue/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 2.4 Move `libs/go/collections/set/set_test.go` to `libs/go/tests/collections/set/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 2.5 Move `libs/go/collections/slices/slices_test.go` to `libs/go/tests/collections/slices/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 2.6 Move `libs/go/collections/sort/sort_test.go` to `libs/go/tests/collections/sort/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 3. Move concurrency module tests
  - [x] 3.1 Move `libs/go/concurrency/async/async_test.go` to `libs/go/tests/concurrency/async/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.2 Move `libs/go/concurrency/atomic/atomic_test.go` to `libs/go/tests/concurrency/atomic/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.3 Move `libs/go/concurrency/channels/channels_test.go` to `libs/go/tests/concurrency/channels/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.4 Move `libs/go/concurrency/errgroup/errgroup_test.go` to `libs/go/tests/concurrency/errgroup/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.5 Move `libs/go/concurrency/once/once_test.go` to `libs/go/tests/concurrency/once/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.6 Move `libs/go/concurrency/pool/pool_test.go` to `libs/go/tests/concurrency/pool/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.7 Move `libs/go/concurrency/syncmap/syncmap_test.go` to `libs/go/tests/concurrency/syncmap/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 3.8 Move `libs/go/concurrency/waitgroup/waitgroup_test.go` to `libs/go/tests/concurrency/waitgroup/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 4. Move events module tests
  - [x] 4.1 Move `libs/go/events/builder/*_test.go` to `libs/go/tests/events/builder/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 4.2 Move `libs/go/events/eventbus/eventbus_test.go` to `libs/go/tests/events/eventbus/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 4.3 Move `libs/go/events/pubsub/pubsub_test.go` to `libs/go/tests/events/pubsub/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 5. Move functional module tests
  - [x] 5.1 Move `libs/go/functional/either/either_test.go` to `libs/go/tests/functional/either/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.2 Move `libs/go/functional/iterator/iterator_test.go` to `libs/go/tests/functional/iterator/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.3 Move `libs/go/functional/lazy/*_test.go` to `libs/go/tests/functional/lazy/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.4 Move `libs/go/functional/option/*_test.go` to `libs/go/tests/functional/option/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.5 Move `libs/go/functional/pipeline/*_test.go` to `libs/go/tests/functional/pipeline/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.6 Move `libs/go/functional/result/*_test.go` to `libs/go/tests/functional/result/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.7 Move `libs/go/functional/stream/*_test.go` to `libs/go/tests/functional/stream/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 5.8 Move `libs/go/functional/tuple/*_test.go` to `libs/go/tests/functional/tuple/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 6. Move grpc, optics, and patterns module tests
  - [x] 6.1 Move `libs/go/grpc/errors/*_test.go` to `libs/go/tests/grpc/errors/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 6.2 Move `libs/go/optics/lens/*_test.go` to `libs/go/tests/optics/lens/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 6.3 Move `libs/go/optics/prism/*_test.go` to `libs/go/tests/optics/prism/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 6.4 Move `libs/go/patterns/registry/*_test.go` to `libs/go/tests/patterns/registry/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 6.5 Move `libs/go/patterns/spec/*_test.go` to `libs/go/tests/patterns/spec/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 7. Move resilience module tests
  - [x] 7.1 Move `libs/go/resilience/*_test.go` (root level) to `libs/go/tests/resilience/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.2 Move `libs/go/resilience/bulkhead/*_test.go` to `libs/go/tests/resilience/bulkhead/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.3 Move `libs/go/resilience/circuitbreaker/*_test.go` to `libs/go/tests/resilience/circuitbreaker/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.4 Move `libs/go/resilience/domain/*_test.go` to `libs/go/tests/resilience/domain/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.5 Move `libs/go/resilience/errors/*_test.go` to `libs/go/tests/resilience/errors/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.6 Move `libs/go/resilience/health/*_test.go` to `libs/go/tests/resilience/health/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.7 Move `libs/go/resilience/rand/*_test.go` to `libs/go/tests/resilience/rand/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.8 Move `libs/go/resilience/ratelimit/*_test.go` to `libs/go/tests/resilience/ratelimit/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.9 Move `libs/go/resilience/retry/*_test.go` to `libs/go/tests/resilience/retry/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.10 Move `libs/go/resilience/shutdown/*_test.go` to `libs/go/tests/resilience/shutdown/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.11 Move `libs/go/resilience/testutil/*_test.go` to `libs/go/tests/resilience/testutil/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 7.12 Move `libs/go/resilience/timeout/*_test.go` to `libs/go/tests/resilience/timeout/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 8. Move server module tests
  - [x] 8.1 Move `libs/go/server/health/*_test.go` to `libs/go/tests/server/health/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 8.2 Move `libs/go/server/shutdown/*_test.go` to `libs/go/tests/server/shutdown/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 8.3 Move `libs/go/server/tracing/*_test.go` to `libs/go/tests/server/tracing/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 9. Move testing and utils module tests
  - [x] 9.1 Move `libs/go/testing/testutil/*_test.go` to `libs/go/tests/testing/testutil/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.2 Move `libs/go/utils/audit/*_test.go` to `libs/go/tests/utils/audit/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.3 Move `libs/go/utils/cache/*_test.go` to `libs/go/tests/utils/cache/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.4 Move `libs/go/utils/codec/*_test.go` to `libs/go/tests/utils/codec/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.5 Move `libs/go/utils/diff/*_test.go` to `libs/go/tests/utils/diff/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.6 Move `libs/go/utils/error/*_test.go` to `libs/go/tests/utils/error/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.7 Move `libs/go/utils/merge/*_test.go` to `libs/go/tests/utils/merge/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.8 Move `libs/go/utils/uuid/*_test.go` to `libs/go/tests/utils/uuid/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.9 Move `libs/go/utils/validated/*_test.go` to `libs/go/tests/utils/validated/`
    - _Requirements: 2.1, 3.1, 3.2_
  - [x] 9.10 Move `libs/go/utils/validator/*_test.go` to `libs/go/tests/utils/validator/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 10. Move validation module tests
  - [x] 10.1 Move `libs/go/validation/*_test.go` to `libs/go/tests/validation/`
    - _Requirements: 2.1, 3.1, 3.2_

- [x] 11. Checkpoint - Verify reorganization
  - Verify no test files remain outside `tests/` directory
  - Verify directory structure mirrors source structure
  - _Requirements: 5.1, 5.2, 5.3_

- [x] 12. Final verification
  - Run `go test` to ensure tests still compile and pass
  - Update any broken import paths if needed
  - _Requirements: 4.1, 4.2, 4.3_

## Notes

- Each move task creates the target directory if it doesn't exist
- Files are moved (not copied) to avoid duplicates
- Existing files in `tests/` are preserved and not overwritten
- The go.mod files in tests/ subdirectories handle module resolution
