package morfx_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicSurfaceDoesNotExposeInternalProjectNames(t *testing.T) {
	forbidden := []string{"file" + "man"}
	roots := []string{
		".github",
		"README.md",
		"cmd",
		"core",
		"docs",
		"mcp",
		"plugins",
		"providers",
		"tools",
	}

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == ".git" || entry.Name() == "bin" || entry.Name() == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !isPublicSurfaceFile(path) {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			lower := strings.ToLower(string(content))
			for _, term := range forbidden {
				if strings.Contains(lower, term) {
					t.Fatalf("%s exposes internal project name %q", path, term)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", root, err)
		}
	}
}

func isPublicSurfaceFile(path string) bool {
	switch filepath.Ext(path) {
	case ".go", ".md", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
