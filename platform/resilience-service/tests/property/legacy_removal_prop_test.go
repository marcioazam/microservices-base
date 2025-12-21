package property

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: resilience-service-architecture-2025, Property 10: Legacy Code Removal
// Validates: Requirements 14.1, 14.2, 14.3

const serviceRoot = "../../"

// TestNoMergeConflictMarkers verifies no git merge conflict markers exist.
func TestNoMergeConflictMarkers(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	goFiles := collectGoFiles(t, serviceRoot)
	if len(goFiles) == 0 {
		t.Skip("No Go files found")
	}

	properties.Property("No merge conflict markers in Go files", prop.ForAll(
		func(idx int) bool {
			if idx >= len(goFiles) {
				return true
			}
			file := goFiles[idx]
			return !containsMergeConflictMarkers(t, file)
		},
		gen.IntRange(0, len(goFiles)-1),
	))

	properties.TestingRun(t)
}

// TestNoBackwardCompatibilityReexports verifies no re-export patterns exist.
func TestNoBackwardCompatibilityReexports(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	goFiles := collectGoFiles(t, serviceRoot+"internal/domain/")
	if len(goFiles) == 0 {
		t.Skip("No domain Go files found")
	}

	properties.Property("No backward compatibility re-export patterns", prop.ForAll(
		func(idx int) bool {
			if idx >= len(goFiles) {
				return true
			}
			file := goFiles[idx]
			// Files that are pure re-exports should be removed
			// Check for "backward compatibility" comments
			return !containsBackwardCompatComment(t, file)
		},
		gen.IntRange(0, len(goFiles)-1),
	))

	properties.TestingRun(t)
}

// TestNoGitkeepInPopulatedDirs verifies no .gitkeep files in directories with content.
func TestNoGitkeepInPopulatedDirs(t *testing.T) {
	dirs := []string{
		serviceRoot + "internal/domain",
		serviceRoot + "internal/policy",
		serviceRoot + "internal/timeout",
		serviceRoot + "cmd/server",
	}

	for _, dir := range dirs {
		gitkeepPath := filepath.Join(dir, ".gitkeep")
		if _, err := os.Stat(gitkeepPath); err == nil {
			// Check if directory has other content
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			hasOtherContent := false
			for _, entry := range entries {
				if entry.Name() != ".gitkeep" {
					hasOtherContent = true
					break
				}
			}
			if hasOtherContent {
				t.Errorf("Directory %s has .gitkeep but also has content", dir)
			}
		}
	}
}

func collectGoFiles(t *testing.T, root string) []string {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Logf("Error walking directory: %v", err)
	}
	return files
}

func containsMergeConflictMarkers(t *testing.T, filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "<<<<<<<") ||
			strings.HasPrefix(line, "=======") ||
			strings.HasPrefix(line, ">>>>>>>") {
			t.Logf("Merge conflict marker found in %s", filePath)
			return true
		}
	}
	return false
}

func containsBackwardCompatComment(t *testing.T, filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		if strings.Contains(line, "backward compatibility") ||
			strings.Contains(line, "backwards compatibility") {
			t.Logf("Backward compatibility pattern found in %s", filePath)
			return true
		}
	}
	return false
}
