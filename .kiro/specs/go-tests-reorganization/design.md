# Design Document

## Overview

This design describes a PowerShell script-based approach to reorganize Go test files from their current scattered locations throughout `libs/go` into a centralized `libs/go/tests` directory with a mirrored folder structure.

The solution uses PowerShell commands to:
1. Discover all `*_test.go` files outside the `tests` directory
2. Calculate target paths maintaining the directory hierarchy
3. Create necessary directories and move files
4. Report results

## Architecture

The reorganization follows a simple file-system operation pattern:

```
libs/go/
├── collections/
│   ├── lru/
│   │   ├── lru.go           (stays)
│   │   └── lru_test.go      (moves to tests/)
│   └── ...
├── resilience/
│   ├── config.go            (stays)
│   └── config_test.go       (moves to tests/)
└── tests/
    ├── collections/
    │   └── lru/
    │       └── lru_test.go  (moved here)
    └── resilience/
        └── config_test.go   (moved here)
```

## Components and Interfaces

### Component 1: Test File Discovery

Responsible for finding all test files that need to be moved.

```powershell
# Pseudocode
function Find-TestFiles {
    param([string]$BasePath)
    
    # Find all *_test.go files
    # Exclude files already in /tests/ directory
    # Return list of file paths with relative paths
}
```

### Component 2: Path Calculator

Calculates the target path for each test file.

```powershell
# Pseudocode
function Get-TargetPath {
    param(
        [string]$SourceFile,
        [string]$BasePath,
        [string]$TestsDir
    )
    
    # Extract relative path from source
    # Construct target path under tests/
    # Return target full path
}
```

### Component 3: File Mover

Handles the actual file movement with directory creation.

```powershell
# Pseudocode
function Move-TestFile {
    param(
        [string]$SourcePath,
        [string]$TargetPath
    )
    
    # Create target directory if not exists
    # Check for conflicts
    # Move file
    # Return success/failure status
}
```

### Component 4: Report Generator

Generates summary of operations performed.

```powershell
# Pseudocode
function Write-Report {
    param(
        [array]$MovedFiles,
        [array]$SkippedFiles,
        [array]$Errors
    )
    
    # Output counts and details
}
```

## Data Models

### FileOperation

Represents a single file move operation:

```
FileOperation {
    SourcePath: string      # Original file location
    TargetPath: string      # Destination location
    Status: enum            # Success, Skipped, Error
    Message: string         # Optional error/skip reason
}
```

### ReorganizationResult

Represents the overall operation result:

```
ReorganizationResult {
    TotalFound: int         # Total test files found
    Moved: int              # Successfully moved
    Skipped: int            # Skipped due to conflicts
    Errors: int             # Failed operations
    Operations: FileOperation[]
}
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Test File Exclusion from Tests Directory

*For any* file already located in `libs/go/tests/`, that file SHALL NOT appear in the list of files to be moved.

**Validates: Requirements 1.2**

### Property 2: Path Transformation Correctness

*For any* test file at path `libs/go/{path}/*_test.go` (where `{path}` does not start with `tests/`), the calculated target path SHALL be `libs/go/tests/{path}/*_test.go`.

**Validates: Requirements 1.3, 2.1**

### Property 3: Move Integrity - Source Removal

*For any* successfully moved test file, the original source file SHALL NOT exist after the move operation completes.

**Validates: Requirements 3.1**

### Property 4: Move Integrity - Content Preservation

*For any* successfully moved test file, the content at the target location SHALL be byte-for-byte identical to the original content before the move.

**Validates: Requirements 3.2**

## Error Handling

### Conflict Detection

When a target file already exists:
1. Log the conflict with source and target paths
2. Add to skipped files list
3. Continue with next file
4. Do NOT overwrite existing files

### Directory Creation Errors

When directory creation fails:
1. Log the error with path and reason
2. Add to errors list
3. Skip the file move
4. Continue with next file

### File Move Errors

When file move fails:
1. Log the error with source, target, and reason
2. Add to errors list
3. Ensure source file is not deleted if move failed
4. Continue with next file

## Testing Strategy

### Manual Verification

Since this is a one-time file reorganization script, testing will be done through:

1. **Dry-run mode**: First run with `-WhatIf` to preview changes
2. **Incremental execution**: Move files in batches to verify correctness
3. **Post-move validation**: Run `go test` in affected directories to ensure tests still work

### Verification Commands

```powershell
# Verify no test files remain outside tests/
Get-ChildItem -Path "libs/go" -Recurse -Filter "*_test.go" | 
    Where-Object { $_.FullName -notmatch "\\tests\\" }

# Verify test structure mirrors source
Get-ChildItem -Path "libs/go/tests" -Recurse -Filter "*_test.go"

# Run tests to verify they still work
cd libs/go && go test ./tests/...
```
