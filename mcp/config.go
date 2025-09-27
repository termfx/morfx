package mcp

import (
	"io"
	"os"
	"path/filepath"
	"time"
)

// Config holds the MCP server configuration
type Config struct {
	// Database
	DatabaseURL string

	// Auto-apply settings
	AutoApplyEnabled   bool
	AutoApplyThreshold float64

	// Staging
	StagingTTL time.Duration

	// Session limits
	MaxStagesPerSession  int
	MaxAppliesPerSession int

	// Safety Configuration
	Safety SafetyConfig

	// Debug
	Debug bool

	// LogWriter allows tests to redirect debug logging; defaults to stderr.
	LogWriter io.Writer
}

// SafetyConfig holds safety-related configuration
type SafetyConfig struct {
	// File operation limits
	MaxFiles     int   `json:"max_files"`      // Max files per operation
	MaxFileSize  int64 `json:"max_file_size"`  // Max size per file (bytes)
	MaxTotalSize int64 `json:"max_total_size"` // Max total operation size (bytes)

	// Confidence validation
	ConfidenceMode   string  `json:"confidence_mode"`    // "global", "per_file", "both"
	PerFileThreshold float64 `json:"per_file_threshold"` // Individual file threshold
	GlobalThreshold  float64 `json:"global_threshold"`   // Overall operation threshold

	// Hash validation
	ValidateFileHashes bool `json:"validate_file_hashes"` // Check files weren't modified externally

	// Atomic writes (always enabled for safety)
	UseFsync bool `json:"use_fsync"` // Use fsync for durability

	// Backup & rollback
	CreateBackups  bool   `json:"create_backups"`  // Create .bak files before writes
	BackupSuffix   string `json:"backup_suffix"`   // Backup file suffix
	TransactionLog bool   `json:"transaction_log"` // Enable transaction logging

	// Concurrency safety
	FileLocking bool          `json:"file_locking"` // Use file locking
	LockTimeout time.Duration `json:"lock_timeout"` // Lock acquisition timeout
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() Config {
	// Use temp directory for database if current dir is not writable, otherwise skip database
	dbPath := "./.morfx/db/morfx.db"
	if !isDirectoryWritable(".") {
		tempDir := os.TempDir()
		if tempDir != "" {
			candidate := filepath.Join(tempDir, "morfx.db")
			if isDirectoryWritable(tempDir) {
				dbPath = candidate
			} else {
				dbPath = "skip"
			}
		} else {
			dbPath = "skip"
		}
	}

	return Config{
		DatabaseURL:          dbPath,
		AutoApplyEnabled:     true,
		AutoApplyThreshold:   0.85,
		StagingTTL:           15 * time.Minute,
		MaxStagesPerSession:  100,
		MaxAppliesPerSession: 10,

		Safety: SafetyConfig{
			// File operation limits
			MaxFiles:     1000,              // Reasonable batch size
			MaxFileSize:  10 * 1024 * 1024,  // 10MB per file
			MaxTotalSize: 100 * 1024 * 1024, // 100MB total

			// Confidence validation
			ConfidenceMode:   "per_file", // Validate each file individually
			PerFileThreshold: 0.70,       // Slightly lower than global for individual files
			GlobalThreshold:  0.85,       // Matches AutoApplyThreshold

			// Hash validation
			ValidateFileHashes: true, // Always validate file integrity

			// Atomic writes (always enabled for safety)
			UseFsync: false, // Performance over extreme durability by default

			// Backup & rollback
			CreateBackups:  false,        // No backups by default (can be overwhelming)
			BackupSuffix:   ".morfx.bak", // Distinctive suffix
			TransactionLog: true,         // Enable transaction logging

			// Concurrency safety
			FileLocking: true,             // Prevent concurrent modifications
			LockTimeout: 30 * time.Second, // Reasonable timeout
		},

		Debug:     false,
		LogWriter: os.Stderr,
	}
}

// isDirectoryWritable checks if a directory is writable
func isDirectoryWritable(path string) bool {
	// Try to create a temporary file
	testFile := filepath.Join(path, ".write_test")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}
