// Package security provides security utilities for the resilience service.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePolicyPath validates that a file path is safe to access.
// It prevents path traversal attacks by ensuring the path stays within
// the specified base directory.
func ValidatePolicyPath(path, basePath string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}

	// Check for null bytes (common attack vector)
	if strings.ContainsRune(path, '\x00') {
		return fmt.Errorf("path contains null bytes")
	}

	// Reject paths with parent directory references BEFORE cleaning
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains parent directory reference")
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// If path is absolute, check it's within base
	if filepath.IsAbs(path) {
		absBase, err := filepath.Abs(basePath)
		if err != nil {
			return fmt.Errorf("resolve absolute base: %w", err)
		}
		if !strings.HasPrefix(cleanPath, absBase) {
			return fmt.Errorf("absolute path '%s' is outside allowed directory '%s'", path, basePath)
		}
		return nil
	}

	// For relative paths, resolve and check
	baseDir := basePath
	if baseDir == "." || baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
	}
	baseDir = filepath.Clean(baseDir)

	// Resolve to absolute paths for comparison
	absPath, err := filepath.Abs(filepath.Join(baseDir, cleanPath))
	if err != nil {
		return fmt.Errorf("resolve absolute path: %w", err)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve absolute base: %w", err)
	}

	// Ensure the path is within the base directory
	if !strings.HasPrefix(absPath, absBase) {
		return fmt.Errorf("path '%s' is outside allowed directory '%s'", path, baseDir)
	}

	return nil
}
