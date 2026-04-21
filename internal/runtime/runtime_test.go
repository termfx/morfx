package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRegistersExpectedLanguages(t *testing.T) {
	t.Parallel()

	rt, err := Build(Config{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	want := []string{"go", "javascript", "typescript", "php", "python"}
	if got := rt.Providers.Languages(); len(got) != len(want) {
		t.Fatalf("Languages() len = %d, want %d (%v)", len(got), len(want), got)
	}

	for _, lang := range want {
		if _, ok := rt.Providers.Get(lang); !ok {
			t.Fatalf("missing language provider %q", lang)
		}
	}
}

func TestBuildCreatesFileProcessor(t *testing.T) {
	t.Parallel()

	rt, err := Build(Config{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if rt.FileProcessor == nil {
		t.Fatal("FileProcessor is nil")
	}
}

func TestBuildCreatesTransactionLogDir(t *testing.T) {
	t.Parallel()

	txDir := filepath.Join(t.TempDir(), "transactions")

	rt, err := Build(Config{TransactionLogDir: txDir})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if rt.FileProcessor == nil {
		t.Fatal("FileProcessor is nil")
	}
	if _, err := os.Stat(txDir); err != nil {
		t.Fatalf("Stat(%q) error = %v", txDir, err)
	}
}
