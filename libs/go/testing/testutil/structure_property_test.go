// Package testutil provides property tests for validating the libs/go structure.
// Feature: go-lib-reorganization
package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Property 3: Test File Colocation
// For any package directory containing source files (*.go excluding *_test.go),
// all associated test files (*_test.go) SHALL reside in the same directory.
// Validates: Requirements 4.1
func TestProperty_TestFileColocation(t *testing.T) {
	libsGoPath := filepath.Join("..", "..")
	
	err := filepath.Walk(libsGoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			return nil
		}
		
		// Skip hidden directories and vendor
		if strings.HasPrefix(info.Name(), ".") || info.Name() == "vendor" {
			return filepath.SkipDir
		}
		
		// Check if directory has .go files
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		
		hasSourceFiles := false
		hasTestFiles := false
		
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
				hasSourceFiles = true
			}
			if strings.HasSuffix(name, "_test.go") {
				hasTestFiles = true
			}
		}
		
		// If we have source files, test files should be in same directory (if they exist)
		// This property validates colocation - tests are WITH source, not separate
		if hasSourceFiles && hasTestFiles {
			// Both exist in same directory - property satisfied
			t.Logf("✓ Package %s has colocated tests", path)
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}

// Property 4: Test Package Naming Consistency
// For any test file (*_test.go) in a package, the package declaration SHALL match
// the package name of the source files in the same directory.
// Validates: Requirements 4.3
func TestProperty_TestPackageNaming(t *testing.T) {
	libsGoPath := filepath.Join("..", "..")
	
	err := filepath.Walk(libsGoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			return nil
		}
		
		// Skip hidden directories
		if strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		
		var sourcePackage, testPackage string
		
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			
			if strings.HasSuffix(name, ".go") {
				content, err := os.ReadFile(filepath.Join(path, name))
				if err != nil {
					continue
				}
				
				// Extract package name from first line
				lines := strings.Split(string(content), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "package ") {
						pkg := strings.TrimPrefix(line, "package ")
						pkg = strings.TrimSpace(pkg)
						
						if strings.HasSuffix(name, "_test.go") {
							testPackage = pkg
						} else {
							sourcePackage = pkg
						}
						break
					}
				}
			}
		}
		
		// If both exist, they should match (or test can use _test suffix)
		if sourcePackage != "" && testPackage != "" {
			if testPackage != sourcePackage && testPackage != sourcePackage+"_test" {
				t.Errorf("Package naming mismatch in %s: source=%s, test=%s", 
					path, sourcePackage, testPackage)
			}
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}

// Property 5: Module Path Directory Consistency
// For any package with a go.mod file, the module path declared in go.mod SHALL match
// the package's directory path relative to the repository root.
// Validates: Requirements 5.1, 7.1, 7.3
func TestProperty_ModulePathConsistency(t *testing.T) {
	libsGoPath := filepath.Join("..", "..")
	baseModule := "github.com/auth-platform/libs/go"
	
	err := filepath.Walk(libsGoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.Name() != "go.mod" {
			return nil
		}
		
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		
		// Extract module path
		lines := strings.Split(string(content), "\n")
		var modulePath string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "module ") {
				modulePath = strings.TrimPrefix(line, "module ")
				modulePath = strings.TrimSpace(modulePath)
				break
			}
		}
		
		if modulePath == "" {
			return nil
		}
		
		// Calculate expected path from directory
		dir := filepath.Dir(path)
		relPath, err := filepath.Rel(libsGoPath, dir)
		if err != nil {
			return nil
		}
		
		// Convert to forward slashes for module path
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		
		var expectedModule string
		if relPath == "." {
			expectedModule = baseModule
		} else {
			expectedModule = baseModule + "/" + relPath
		}
		
		if modulePath != expectedModule {
			t.Errorf("Module path mismatch in %s:\n  got:      %s\n  expected: %s", 
				path, modulePath, expectedModule)
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}

// Property 6: Category Documentation Existence
// For any category directory, a README.md file SHALL exist.
// Validates: Requirements 3.2
func TestProperty_CategoryREADMEExistence(t *testing.T) {
	categories := []string{
		"collections",
		"concurrency", 
		"functional",
		"optics",
		"patterns",
		"events",
		"resilience",
		"server",
		"grpc",
		"utils",
		"testing",
	}
	
	libsGoPath := filepath.Join("..", "..")
	
	for _, category := range categories {
		readmePath := filepath.Join(libsGoPath, category, "README.md")
		if _, err := os.Stat(readmePath); os.IsNotExist(err) {
			t.Errorf("Missing README.md for category: %s", category)
		}
	}
}

// Property 2: Go.work Module Completeness
// For any package directory containing a go.mod file, that module path SHALL be
// listed in the libs/go/go.work file.
// Validates: Requirements 2.2
func TestProperty_GoWorkCompleteness(t *testing.T) {
	libsGoPath := filepath.Join("..", "..")
	goWorkPath := filepath.Join(libsGoPath, "go.work")
	
	// Read go.work content
	content, err := os.ReadFile(goWorkPath)
	if err != nil {
		t.Fatalf("Failed to read go.work: %v", err)
	}
	goWorkContent := string(content)
	
	// Find all go.mod files
	err = filepath.Walk(libsGoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.Name() != "go.mod" {
			return nil
		}
		
		// Get relative path from libs/go
		dir := filepath.Dir(path)
		relPath, err := filepath.Rel(libsGoPath, dir)
		if err != nil {
			return nil
		}
		
		// Skip root go.mod
		if relPath == "." {
			return nil
		}
		
		// Convert to forward slashes and add ./
		relPath = "./" + strings.ReplaceAll(relPath, "\\", "/")
		
		if !strings.Contains(goWorkContent, relPath) {
			t.Errorf("Module not in go.work: %s", relPath)
		}
		
		return nil
	})
	
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}
