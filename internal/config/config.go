package config

import (
	"os"
	"strconv"
)

// Config holds the application's configuration.
type Config struct {
	EncryptionMode      string
	MasterKey           string
	EncryptionAlgo      string
	ActiveKeyVersion    int
	WALAutoCheckpointMB int
	RetentionRuns       int
}

// LoadConfig loads configuration from environment variables.
func LoadConfig() *Config {
	cfg := &Config{
		EncryptionMode:      os.Getenv("MORFX_ENCRYPTION_MODE"),
		MasterKey:           os.Getenv("MORFX_MASTER_KEY"),
		EncryptionAlgo:      os.Getenv("MORFX_ENCRYPTION_ALGO"),
		ActiveKeyVersion:    0,   // Default value
		WALAutoCheckpointMB: 128, // Default value
		RetentionRuns:       20,  // Default value
	}

	if cfg.EncryptionMode == "" {
		cfg.EncryptionMode = "auto"
	}
	if cfg.EncryptionAlgo == "" {
		cfg.EncryptionAlgo = "xchacha20poly1305"
	}

	if keyVersionStr := os.Getenv("MORFX_ACTIVE_KEY_VERSION"); keyVersionStr != "" {
		if keyVersion, err := strconv.Atoi(keyVersionStr); err == nil && keyVersion >= 0 {
			cfg.ActiveKeyVersion = keyVersion
		}
	}

	if walCheckpointStr := os.Getenv("MORFX_DB_WAL_AUTOCHECKPOINT_MB"); walCheckpointStr != "" {
		if walCheckpoint, err := strconv.Atoi(walCheckpointStr); err == nil && walCheckpoint > 0 {
			cfg.WALAutoCheckpointMB = walCheckpoint
		}
	}

	if retentionRunsStr := os.Getenv("MORFX_DB_RETENTION_RUNS"); retentionRunsStr != "" {
		if retentionRuns, err := strconv.Atoi(retentionRunsStr); err == nil && retentionRuns >= 0 {
			cfg.RetentionRuns = retentionRuns
		}
	}

	return cfg
}
