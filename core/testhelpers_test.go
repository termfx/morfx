package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func setUnixModeIfSupported(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Logf("Skipping chmod(%s, %04o) on Windows", path, mode)
		return
	}

	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("Failed to set file mode for %s: %v", path, err)
	}
}

func assertUnixModeIfSupported(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat %s: %v", path, err)
	}

	if runtime.GOOS == "windows" {
		t.Logf("Skipping mode assertion for %s on Windows; got %v", path, info.Mode().Perm())
		return
	}

	if got := info.Mode().Perm(); got != want {
		t.Errorf("Permissions for %s = %v, want %v", path, got, want)
	}
}

func containsFilePath(paths []string, want string) bool {
	want = filepath.Clean(want)
	for _, path := range paths {
		if filepath.Clean(path) == want {
			return true
		}
	}
	return false
}
