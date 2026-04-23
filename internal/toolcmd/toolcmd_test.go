package toolcmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatConfidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{name: "clamps low values", score: -0.25, want: "░░░░░░░░░░"},
		{name: "renders zero", score: 0, want: "░░░░░░░░░░"},
		{name: "renders half", score: 0.5, want: "█████░░░░░"},
		{name: "renders full", score: 1, want: "██████████"},
		{name: "clamps high values", score: 1.4, want: "██████████"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := FormatConfidence(tt.score); got != tt.want {
				t.Fatalf("FormatConfidence(%v) = %q, want %q", tt.score, got, tt.want)
			}
		})
	}
}

func TestWriteModifiedSource(t *testing.T) {
	t.Parallel()

	t.Run("skips when not from file or unchanged", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "sample.go")
		original := "package main\n"

		if err := os.WriteFile(path, []byte(original), 0o640); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		wrote, err := WriteModifiedSource(path, false, original, "package main\n// updated\n", 0o640)
		if err != nil {
			t.Fatalf("WriteModifiedSource() error = %v", err)
		}
		if wrote {
			t.Fatal("WriteModifiedSource() wrote file for non-file source")
		}

		wrote, err = WriteModifiedSource(path, true, original, original, 0o640)
		if err != nil {
			t.Fatalf("WriteModifiedSource() error = %v", err)
		}
		if wrote {
			t.Fatal("WriteModifiedSource() wrote file for unchanged content")
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(data) != original {
			t.Fatalf("file content changed unexpectedly: %q", string(data))
		}
	})

	t.Run("writes modified content", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "sample.go")
		original := "package main\n"
		modified := "package main\n// updated\n"

		if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		wrote, err := WriteModifiedSource(path, true, original, modified, 0o600)
		if err != nil {
			t.Fatalf("WriteModifiedSource() error = %v", err)
		}
		if !wrote {
			t.Fatal("WriteModifiedSource() did not write modified content")
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(data) != modified {
			t.Fatalf("file content = %q, want %q", string(data), modified)
		}
	})
}
