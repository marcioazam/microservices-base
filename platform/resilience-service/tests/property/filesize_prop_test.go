package property

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

const maxNonBlankLines = 400

// **Feature: resilience-service-state-of-art-2025, Property 7: File Size Compliance**
// **Validates: Requirements 9.1**
func TestProperty_FileSizeCompliance(t *testing.T) {
	t.Run("all_source_files_under_400_lines", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")

			rootDir := filepath.Join("..", "..")

			dirsToCheck := []string{
				filepath.Join(rootDir, "internal"),
				filepath.Join(rootDir, "cmd"),
			}

			for _, dir := range dirsToCheck {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					continue
				}

				err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					if info.IsDir() {
						return nil
					}
					if !strings.HasSuffix(info.Name(), ".go") {
						return nil
					}
					if strings.HasSuffix(info.Name(), "_test.go") {
						return nil
					}

					nonBlankLines, err := countNonBlankLines(path)
					if err != nil {
						return nil
					}

					if nonBlankLines > maxNonBlankLines {
						t.Errorf("file %s has %d non-blank lines (max %d)", path, nonBlankLines, maxNonBlankLines)
					}

					return nil
				})

				if err != nil {
					t.Fatalf("walk error: %v", err)
				}
			}
		})
	})

	t.Run("test_files_under_400_lines", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			_ = rapid.IntRange(1, 1).Draw(t, "iteration")

			testsDir := filepath.Join("..", "..")

			err := filepath.Walk(testsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				if !strings.HasSuffix(info.Name(), "_test.go") {
					return nil
				}

				nonBlankLines, err := countNonBlankLines(path)
				if err != nil {
					return nil
				}

				if nonBlankLines > maxNonBlankLines {
					t.Errorf("test file %s has %d non-blank lines (max %d)", path, nonBlankLines, maxNonBlankLines)
				}

				return nil
			})

			if err != nil {
				t.Fatalf("walk error: %v", err)
			}
		})
	})
}

func countNonBlankLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}

	return count, scanner.Err()
}
