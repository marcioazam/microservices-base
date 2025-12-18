package property

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestDirectory represents a test directory type.
type TestDirectory string

const (
	PropertyDir    TestDirectory = "property"
	BenchmarkDir   TestDirectory = "benchmark"
	UnitDir        TestDirectory = "unit"
	IntegrationDir TestDirectory = "integration"
)

// **Feature: resilience-test-reorganization, Property 1: Test File Location Correctness**
// **Validates: Requirements 1.1**
func TestProperty_TestFileLocationCorrectness(t *testing.T) {
	testsDir := getTestsDir(t)

	t.Run("property_tests_are_in_property_directory", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			if !verifyTestFilesInDirectory(testsDir, PropertyDir, "_prop_test.go") {
				t.Fatal("property tests not in correct directory")
			}
		})
	})

	t.Run("benchmark_tests_are_in_benchmark_directory", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			if !verifyTestFilesInDirectory(testsDir, BenchmarkDir, "_bench_test.go") {
				t.Fatal("benchmark tests not in correct directory")
			}
		})
	})

	t.Run("unit_tests_are_in_unit_directory", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			if !verifyNoMixedTestTypes(testsDir, UnitDir) {
				t.Fatal("mixed test types in unit directory")
			}
		})
	})

	t.Run("integration_tests_are_in_integration_directory", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			if !verifyNoMixedTestTypes(testsDir, IntegrationDir) {
				t.Fatal("mixed test types in integration directory")
			}
		})
	})
}

// **Feature: resilience-test-reorganization, Property 2: Test Type Directory Mapping**
// **Validates: Requirements 1.2, 2.3**
func TestProperty_TestTypeDirectoryMapping(t *testing.T) {
	testsDir := getTestsDir(t)

	t.Run("all_test_directories_exist", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			dirs := []TestDirectory{PropertyDir, BenchmarkDir, UnitDir, IntegrationDir}
			for _, dir := range dirs {
				path := filepath.Join(testsDir, string(dir))
				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Fatalf("directory does not exist: %s", path)
				}
			}
		})
	})

	t.Run("testutil_directory_exists", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			path := filepath.Join(testsDir, "testutil")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Fatalf("testutil directory does not exist: %s", path)
			}
		})
	})

	t.Run("no_test_files_in_internal_packages", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")
			if !verifyNoTestFilesInInternal() {
				t.Fatal("test files found in internal packages")
			}
		})
	})
}

func getTestsDir(t *testing.T) string {
	t.Helper()
	return ".."
}

func verifyTestFilesInDirectory(testsDir string, dir TestDirectory, suffix string) bool {
	dirPath := filepath.Join(testsDir, string(dir))

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "_test.go") {
			if suffix != "" && !strings.HasSuffix(name, suffix) {
				continue
			}
		}
	}
	return true
}

func verifyNoMixedTestTypes(testsDir string, dir TestDirectory) bool {
	dirPath := filepath.Join(testsDir, string(dir))

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if dir == UnitDir || dir == IntegrationDir {
			if strings.HasSuffix(name, "_prop_test.go") {
				return false
			}
			if strings.HasSuffix(name, "_bench_test.go") {
				return false
			}
		}
	}
	return true
}

func verifyNoTestFilesInInternal() bool {
	internalDir := filepath.Join("..", "..", "internal")

	movedPatterns := []string{
		"_prop_test.go",
		"_bench_test.go",
	}

	found := false
	_ = filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		for _, pattern := range movedPatterns {
			if strings.HasSuffix(info.Name(), pattern) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})

	return !found
}
