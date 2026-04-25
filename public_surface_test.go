package morfx_test

import (
	"os"
	"path/filepath"
	"regexp"
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

func TestPublicReleaseNotesMatchCurrentTimeline(t *testing.T) {
	allowed := map[string]struct{}{
		"docs/release-notes-v0.1.0.md": {},
		"docs/release-notes-v0.2.0.md": {},
		"docs/release-notes-v0.3.0.md": {},
		"docs/release-notes-v0.4.0.md": {},
	}
	pattern := regexp.MustCompile(`^release-notes-v[0-9]+\.[0-9]+\.[0-9]+\.md$`)

	entries, err := os.ReadDir("docs")
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		path := filepath.ToSlash(filepath.Join("docs", entry.Name()))
		if _, ok := allowed[path]; !ok {
			t.Fatalf("%s is not part of the current public release timeline", path)
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
