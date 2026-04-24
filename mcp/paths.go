package mcp

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/oxhq/morfx/internal/securefs"
)

func defaultStateDir() string {
	candidates := []string{}

	if override := strings.TrimSpace(os.Getenv("MORFX_STATE_DIR")); override != "" {
		candidates = append(candidates, override)
	}

	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		candidates = append(candidates, filepath.Join(configDir, "morfx"))
	}

	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".morfx"))
	}

	if tempDir := os.TempDir(); tempDir != "" {
		candidates = append(candidates, filepath.Join(tempDir, "morfx"))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}

		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		if ensureWritableDir(candidate) {
			return candidate
		}
	}

	return ""
}

func defaultDatabaseURL() string {
	stateDir := defaultStateDir()
	if stateDir == "" {
		return "skip"
	}

	dbDir := filepath.Join(stateDir, "db")
	if !ensureWritableDir(dbDir) {
		return "skip"
	}

	return filepath.Join(dbDir, "morfx.db")
}

func defaultTransactionLogDir() string {
	stateDir := defaultStateDir()
	if stateDir == "" {
		return ""
	}

	txDir := filepath.Join(stateDir, "transactions")
	if !ensureWritableDir(txDir) {
		return ""
	}

	return txDir
}

func ensureWritableDir(path string) bool {
	if path == "" {
		return false
	}

	if err := securefs.MkdirAll(path, 0o700); err != nil {
		return false
	}

	return isDirectoryWritable(path)
}
