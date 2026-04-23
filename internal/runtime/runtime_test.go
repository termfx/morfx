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

func TestBuildLeavesBuiltinTransactionLogDirWhenEmpty(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", tmpDir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	rt, err := Build(Config{})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if rt.FileProcessor == nil {
		t.Fatal("FileProcessor is nil")
	}

	txDir := filepath.Join(tmpDir, ".morfx", "transactions")
	if _, statErr := os.Stat(txDir); !os.IsNotExist(statErr) {
		t.Fatalf("empty-config Build() created %q; want no cwd-local transaction dir", txDir)
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
