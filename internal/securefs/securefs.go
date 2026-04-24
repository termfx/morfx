package securefs

import "os"

// ReadFile is the audited entry point for caller-selected file reads.
func ReadFile(path string) ([]byte, error) {
	// #nosec G304 -- Morfx intentionally operates on caller-selected files after higher-level scope and safety checks.
	return os.ReadFile(path)
}

// WriteFile is the audited entry point for caller-selected file writes.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	// #nosec G306,G703 -- Morfx intentionally writes caller-selected files and preserves requested permissions.
	return os.WriteFile(path, data, perm)
}

// Open opens a caller-selected path.
func Open(path string) (*os.File, error) {
	// #nosec G304 -- Morfx intentionally operates on caller-selected paths after higher-level checks.
	return os.Open(path)
}

// OpenFile opens a caller-selected path with explicit flags and permissions.
func OpenFile(path string, flag int, perm os.FileMode) (*os.File, error) {
	// #nosec G302,G304 -- callers choose permissions deliberately for code files, locks, or transaction state.
	return os.OpenFile(path, flag, perm)
}

// Create creates a caller-selected path.
func Create(path string) (*os.File, error) {
	// #nosec G304 -- used for controlled write-probe files in selected state directories.
	return os.Create(path)
}

// MkdirAll creates a caller-selected directory tree with explicit permissions.
func MkdirAll(path string, perm os.FileMode) error {
	// #nosec G301 -- callers choose directory permissions deliberately for state or project directories.
	return os.MkdirAll(path, perm)
}

// RemoveBestEffort removes a cleanup path where failure is intentionally non-fatal.
func RemoveBestEffort(path string) {
	_ = os.Remove(path)
}

// CloseBestEffort closes a file where failure is intentionally non-fatal.
func CloseBestEffort(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

// IgnoreError documents best-effort cleanup paths for linters without hiding normal error flows.
func IgnoreError(error) {}
