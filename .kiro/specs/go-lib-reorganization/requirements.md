# Requirements Document

## Introduction

This specification defines the architectural reorganization of the `libs/go` shared library collection. The goal is to organize ~48 packages into logical domain categories following Go best practices and community standards, improving discoverability and maintainability while preserving Go's idiomatic test organization (tests alongside source).

## Glossary

- **Domain Category**: A logical grouping of related packages by functionality (e.g., collections, concurrency, functional)
- **Go Workspace**: A go.work file that coordinates multiple Go modules for local development
- **Import Path**: The full module path used to import a package (e.g., `github.com/auth-platform/libs/go/collections/slices`)
- **Idiomatic Go**: Following Go community conventions and best practices

## Requirements

### Requirement 1: Domain-Based Organization

**User Story:** As a developer, I want packages organized by domain/functionality, so that I can easily discover and use related packages.

#### Acceptance Criteria

1. WHEN a developer browses libs/go THEN the system SHALL display packages grouped into domain categories (collections, concurrency, functional, optics, patterns, events, resilience, server, grpc, utils, testing)
2. WHEN a developer needs a collection utility THEN the system SHALL provide all collection packages under `collections/` directory
3. WHEN a developer needs concurrency primitives THEN the system SHALL provide all concurrency packages under `concurrency/` directory
4. WHEN a developer needs functional types THEN the system SHALL provide all functional packages under `functional/` directory

### Requirement 2: Go Workspace Configuration

**User Story:** As a developer, I want a go.work file configured, so that I can work with multiple modules simultaneously during local development.

#### Acceptance Criteria

1. WHEN a developer clones the repository THEN the system SHALL provide a go.work file at libs/go/go.work
2. WHEN the go.work file is present THEN the system SHALL include all package modules in the workspace
3. WHEN a developer runs `go build` in the workspace THEN the system SHALL resolve all local module dependencies

### Requirement 3: Documentation Structure

**User Story:** As a developer, I want comprehensive documentation, so that I can understand the library organization and usage.

#### Acceptance Criteria

1. WHEN a developer opens libs/go THEN the system SHALL provide a README.md with package index organized by category
2. WHEN a developer opens a domain category directory THEN the system SHALL provide a README.md explaining the category
3. WHEN a developer opens a package directory THEN the system SHALL provide a README.md with usage examples

### Requirement 4: Test Organization (Go Idiomatic)

**User Story:** As a developer, I want tests organized following Go conventions, so that the codebase follows community standards.

#### Acceptance Criteria

1. WHEN a package contains source code THEN the system SHALL keep test files (*_test.go) in the same directory as source files
2. WHEN running `go test ./...` THEN the system SHALL execute all tests successfully
3. WHEN a test needs to access unexported functions THEN the system SHALL use the same package name (not _test suffix)

### Requirement 5: Import Path Updates

**User Story:** As a developer, I want consistent import paths, so that I can import packages using the new domain-based structure.

#### Acceptance Criteria

1. WHEN a package is moved to a new location THEN the system SHALL update its go.mod module path
2. WHEN a consumer imports a reorganized package THEN the system SHALL provide the new import path
3. WHEN the resilience-service imports libs/go packages THEN the system SHALL update all import statements and replace directives

### Requirement 6: Backward Compatibility Documentation

**User Story:** As a developer, I want migration documentation, so that I can update existing code to use new import paths.

#### Acceptance Criteria

1. WHEN a package location changes THEN the system SHALL document the old and new import paths
2. WHEN a developer needs to migrate THEN the system SHALL provide a migration guide in the README

### Requirement 7: Module Configuration

**User Story:** As a developer, I want each package to be a proper Go module, so that it can be imported independently.

#### Acceptance Criteria

1. WHEN a package exists THEN the system SHALL have a go.mod file with correct module path
2. WHEN a package has dependencies THEN the system SHALL declare them in go.mod
3. WHEN packages are reorganized THEN the system SHALL update module paths to reflect new locations

### Requirement 8: Build Validation

**User Story:** As a developer, I want the reorganized library to build successfully, so that I can use it in my projects.

#### Acceptance Criteria

1. WHEN running `go build ./...` in libs/go THEN the system SHALL complete without errors
2. WHEN running `go test ./...` in libs/go THEN the system SHALL pass all tests
3. WHEN running `go build ./...` in platform/resilience-service THEN the system SHALL complete without errors after import updates
