package util

import (
	"strings"
	"testing"
)

func TestUnifiedDiff(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		to       string
		path     string
		context  int
		expected string
	}{
		{
			name:     "no changes",
			from:     "line1\nline2\nline3",
			to:       "line1\nline2\nline3",
			path:     "test.go",
			context:  3,
			expected: "",
		},
		{
			name:    "simple replacement",
			from:    "line1\nline2\nline3",
			to:      "line1\nmodified\nline3",
			path:    "test.go",
			context: 3,
			expected: `--- a/test.go
+++ b/test.go
@@ -1,3 +1,3 @@
 line1
-line2
+modified
 line3
`,
		},
		{
			name:    "addition",
			from:    "line1\nline3",
			to:      "line1\nline2\nline3",
			path:    "test.go",
			context: 3,
			expected: `--- a/test.go
+++ b/test.go
@@ -1,2 +1,3 @@
 line1
+line2
 line3
`,
		},
		{
			name:    "deletion",
			from:    "line1\nline2\nline3",
			to:      "line1\nline3",
			path:    "test.go",
			context: 3,
			expected: `--- a/test.go
+++ b/test.go
@@ -1,3 +1,2 @@
 line1
-line2
 line3
`,
		},
		{
			name:    "stable headers without ANSI colors",
			from:    "old",
			to:      "new",
			path:    "example.txt",
			context: 1,
			expected: `--- a/example.txt
+++ b/example.txt
@@ -1,1 +1,1 @@
-old
+new
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnifiedDiff(tt.from, tt.to, tt.path, tt.context)
			if result != tt.expected {
				t.Errorf("UnifiedDiff() = %q, want %q", result, tt.expected)
				// Print line by line for easier debugging
				resultLines := strings.Split(result, "\n")
				expectedLines := strings.Split(tt.expected, "\n")
				t.Logf("Result lines: %d, Expected lines: %d", len(resultLines), len(expectedLines))
				for i := 0; i < len(resultLines) || i < len(expectedLines); i++ {
					var rLine, eLine string
					if i < len(resultLines) {
						rLine = resultLines[i]
					}
					if i < len(expectedLines) {
						eLine = expectedLines[i]
					}
					if rLine != eLine {
						t.Logf("Line %d: got %q, want %q", i, rLine, eLine)
					}
				}
			}
		})
	}
}

func TestGenerateDiffHunks(t *testing.T) {
	tests := []struct {
		name      string
		fromLines []string
		toLines   []string
		context   int
		wantCount int
	}{
		{
			name:      "no changes",
			fromLines: []string{"line1", "line2", "line3"},
			toLines:   []string{"line1", "line2", "line3"},
			context:   3,
			wantCount: 0,
		},
		{
			name:      "single change",
			fromLines: []string{"line1", "line2", "line3"},
			toLines:   []string{"line1", "modified", "line3"},
			context:   3,
			wantCount: 1,
		},
		{
			name:      "multiple changes",
			fromLines: []string{"line1", "line2", "line3", "line4", "line5"},
			toLines:   []string{"modified1", "line2", "line3", "line4", "modified5"},
			context:   1,
			wantCount: 1, // Should be merged into one hunk with context
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hunks := generateDiffHunks(tt.fromLines, tt.toLines, tt.context)
			if len(hunks) != tt.wantCount {
				t.Errorf("generateDiffHunks() returned %d hunks, want %d", len(hunks), tt.wantCount)
				for i, hunk := range hunks {
					t.Logf("Hunk %d:\n%s", i, hunk)
				}
			}
		})
	}
}
