package morfx_test

import (
	"os"
	"os/exec"
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
	expected := map[string]struct{}{
		"docs/release-notes-v0.2.0.md": {},
		"docs/release-notes-v0.3.0.md": {},
		"docs/release-notes-v0.4.0.md": {},
		"docs/release-notes-v0.5.0.md": {},
	}
	pattern := regexp.MustCompile(`^release-notes-v[0-9]+\.[0-9]+\.[0-9]+\.md$`)

	entries, err := os.ReadDir("docs")
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	actual := make(map[string]struct{}, len(expected))
	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		path := filepath.ToSlash(filepath.Join("docs", entry.Name()))
		actual[path] = struct{}{}
		if _, ok := expected[path]; !ok {
			t.Fatalf("%s is not part of the current public release timeline", path)
		}
	}
	for path := range expected {
		if _, ok := actual[path]; !ok {
			t.Fatalf("%s is required in the current public release timeline", path)
		}
	}
}

func TestPublicDocsMatchEngineManagedStaging(t *testing.T) {
	checks := []struct {
		path      string
		required  []string
		forbidden []string
	}{
		{
			path: "README.md",
			required: []string{
				"Engine-managed staged changes",
				"morfx mcp --stage-dir ./my-stages",
			},
			forbidden: []string{
				"Two-phase commit with SQLite audit trail",
				"morfx mcp --db ./my.db",
			},
		},
		{
			path: "docs/standalone-tools.md",
			required: []string{
				"apply [--root path] [--stage-dir path]",
				"Apply pending engine-managed staged changes from the stage store.",
			},
			forbidden: []string{
				"apply --db",
				"Apply staged transformations stored in the Morfx database.",
				"SQLite/Turso DSN",
			},
		},
		{
			path: "CHANGELOG.md",
			required: []string{
				"## v0.5.0",
			},
		},
		{
			path: "docs/release-notes-v0.5.0.md",
			required: []string{
				"# Morfx v0.5.0",
				"engine",
			},
		},
	}

	for _, check := range checks {
		content, err := os.ReadFile(check.path)
		if err != nil {
			t.Fatalf("read %s: %v", check.path, err)
		}
		text := string(content)
		for _, required := range check.required {
			if !strings.Contains(text, required) {
				t.Fatalf("%s must contain %q", check.path, required)
			}
		}
		for _, forbidden := range check.forbidden {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s must not contain %q", check.path, forbidden)
			}
		}
	}
}

func TestCIWorkflowValidatesDevelopBranch(t *testing.T) {
	content, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("read CI workflow: %v", err)
	}
	text := string(content)

	required := []string{
		"push:\n    branches: [main, develop]",
		"pull_request:\n    branches: [main, develop]",
	}
	for _, snippet := range required {
		if !strings.Contains(text, snippet) {
			t.Fatalf("CI workflow must contain %q", snippet)
		}
	}
}

func TestTrackedPublicSurfaceExcludesAgentFacingPaths(t *testing.T) {
	cmd := exec.Command("git", "ls-files", "--", "docs/superpowers", "plugins/morfx-codex")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list tracked files: %v\n%s", err, output)
	}

	var present []string
	for _, path := range strings.Fields(strings.TrimSpace(string(output))) {
		if _, err := os.Stat(path); err == nil {
			present = append(present, path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat tracked path %s: %v", path, err)
		}
	}

	if len(present) > 0 {
		t.Fatalf("tracked agent-facing paths must not be part of the public surface: %v", present)
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
