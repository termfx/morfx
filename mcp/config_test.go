package mcp

import (
	"slices"
	"testing"
	"time"
)

// TestDefaultConfig tests the default configuration generation
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test basic defaults
	if config.AutoApplyEnabled != true {
		t.Error("AutoApplyEnabled should be true by default")
	}

	if config.AutoApplyThreshold != 0.85 {
		t.Errorf("AutoApplyThreshold should be 0.85, got %f", config.AutoApplyThreshold)
	}

	if config.StagingTTL != 15*time.Minute {
		t.Errorf("StagingTTL should be 15 minutes, got %v", config.StagingTTL)
	}

	if config.MaxStagesPerSession != 100 {
		t.Errorf("MaxStagesPerSession should be 100, got %d", config.MaxStagesPerSession)
	}

	if config.MaxAppliesPerSession != 10 {
		t.Errorf("MaxAppliesPerSession should be 10, got %d", config.MaxAppliesPerSession)
	}

	// Test safety config defaults
	safety := config.Safety
	if safety.MaxFiles != 1000 {
		t.Errorf("MaxFiles should be 1000, got %d", safety.MaxFiles)
	}

	if safety.MaxFileSize != 10*1024*1024 {
		t.Errorf("MaxFileSize should be 10MB, got %d", safety.MaxFileSize)
	}

	if safety.MaxTotalSize != 100*1024*1024 {
		t.Errorf("MaxTotalSize should be 100MB, got %d", safety.MaxTotalSize)
	}

	if safety.ConfidenceMode != "per_file" {
		t.Errorf("ConfidenceMode should be 'per_file', got %s", safety.ConfidenceMode)
	}

	if safety.PerFileThreshold != 0.70 {
		t.Errorf("PerFileThreshold should be 0.70, got %f", safety.PerFileThreshold)
	}

	if safety.GlobalThreshold != 0.85 {
		t.Errorf("GlobalThreshold should be 0.85, got %f", safety.GlobalThreshold)
	}

	if !safety.ValidateFileHashes {
		t.Error("ValidateFileHashes should be true by default")
	}

	if !safety.AtomicWrites {
		t.Error("AtomicWrites should be true by default")
	}

	if safety.UseFsync {
		t.Error("UseFsync should be false by default for performance")
	}

	if safety.CreateBackups {
		t.Error("CreateBackups should be false by default")
	}

	if safety.BackupSuffix != ".morfx.bak" {
		t.Errorf("BackupSuffix should be '.morfx.bak', got %s", safety.BackupSuffix)
	}

	if !safety.TransactionLog {
		t.Error("TransactionLog should be true by default")
	}

	if !safety.FileLocking {
		t.Error("FileLocking should be true by default")
	}

	if safety.LockTimeout != 30*time.Second {
		t.Errorf("LockTimeout should be 30 seconds, got %v", safety.LockTimeout)
	}

	if config.Debug {
		t.Error("Debug should be false by default")
	}
}

// TestConfigDatabaseURLLogic tests database URL determination logic
func TestConfigDatabaseURLLogic(t *testing.T) {
	config := DefaultConfig()

	// The database URL should be set to something reasonable
	if config.DatabaseURL == "" {
		t.Error("DatabaseURL should not be empty")
	}

	// Should contain a valid database path or "skip"
	validURLs := []string{"./.morfx/db/morfx.db", "/tmp/morfx.db", "skip"}
	validURL := slices.Contains(validURLs, config.DatabaseURL)

	if !validURL {
		t.Errorf("DatabaseURL %s is not one of the expected values", config.DatabaseURL)
	}
}

// TestIsDirectoryWritable tests the directory writability check
func TestIsDirectoryWritable(t *testing.T) {
	// Test current directory (should usually be writable in test environment)
	if !isDirectoryWritable(".") {
		t.Log("Current directory is not writable - this might be expected in some environments")
	}

	// Test a directory that definitely doesn't exist
	if isDirectoryWritable("/nonexistent/directory/path") {
		t.Error("Should return false for non-existent directory")
	}

	// Test /tmp if it exists (usually writable)
	tmpWritable := isDirectoryWritable("/tmp")
	t.Logf("Temp directory writable: %v", tmpWritable)
}

// TestSafetyConfigValidation tests safety configuration validation
func TestSafetyConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config SafetyConfig
		valid  bool
	}{
		{
			name: "valid_config",
			config: SafetyConfig{
				MaxFiles:         100,
				MaxFileSize:      1024 * 1024,
				MaxTotalSize:     10 * 1024 * 1024,
				ConfidenceMode:   "per_file",
				PerFileThreshold: 0.7,
				GlobalThreshold:  0.8,
				AtomicWrites:     true,
				FileLocking:      true,
				LockTimeout:      30 * time.Second,
			},
			valid: true,
		},
		{
			name: "zero_values",
			config: SafetyConfig{
				MaxFiles:         0,
				MaxFileSize:      0,
				MaxTotalSize:     0,
				ConfidenceMode:   "",
				PerFileThreshold: 0,
				GlobalThreshold:  0,
			},
			valid: true, // Zero values are technically valid, just not useful
		},
		{
			name: "negative_timeout",
			config: SafetyConfig{
				LockTimeout: -1 * time.Second,
			},
			valid: true, // Negative timeout is handled by the system
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since there's no explicit validation function,
			// we test that the config can be used to create a server
			config := DefaultConfig()
			config.Safety = tt.config
			config.DatabaseURL = "skip"

			server, err := NewStdioServer(config)
			if tt.valid && err != nil {
				t.Errorf("Expected valid config to work, got error: %v", err)
			}
			if server != nil {
				server.Close()
			}
		})
	}
}

// TestConfigWithDifferentValues tests config with various values
func TestConfigWithDifferentValues(t *testing.T) {
	tests := []struct {
		name       string
		modifier   func(*Config)
		shouldWork bool
	}{
		{
			name: "debug_enabled",
			modifier: func(c *Config) {
				c.Debug = true
			},
			shouldWork: true,
		},
		{
			name: "large_staging_ttl",
			modifier: func(c *Config) {
				c.StagingTTL = 24 * time.Hour
			},
			shouldWork: true,
		},
		{
			name: "zero_staging_ttl",
			modifier: func(c *Config) {
				c.StagingTTL = 0
			},
			shouldWork: true,
		},
		{
			name: "high_thresholds",
			modifier: func(c *Config) {
				c.AutoApplyThreshold = 0.99
				c.Safety.GlobalThreshold = 0.95
				c.Safety.PerFileThreshold = 0.90
			},
			shouldWork: true,
		},
		{
			name: "low_thresholds",
			modifier: func(c *Config) {
				c.AutoApplyThreshold = 0.1
				c.Safety.GlobalThreshold = 0.1
				c.Safety.PerFileThreshold = 0.1
			},
			shouldWork: true,
		},
		{
			name: "auto_apply_disabled",
			modifier: func(c *Config) {
				c.AutoApplyEnabled = false
			},
			shouldWork: true,
		},
		{
			name: "max_limits_high",
			modifier: func(c *Config) {
				c.MaxStagesPerSession = 10000
				c.MaxAppliesPerSession = 1000
			},
			shouldWork: true,
		},
		{
			name: "max_limits_low",
			modifier: func(c *Config) {
				c.MaxStagesPerSession = 1
				c.MaxAppliesPerSession = 1
			},
			shouldWork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DatabaseURL = "skip"
			tt.modifier(&config)

			server, err := NewStdioServer(config)

			if tt.shouldWork && err != nil {
				t.Errorf("Expected config to work, got error: %v", err)
			} else if !tt.shouldWork && err == nil {
				t.Error("Expected config to fail but it worked")
			}

			if server != nil {
				server.Close()
			}
		})
	}
}

// TestSafetyConfigModes tests different safety configuration modes
func TestSafetyConfigModes(t *testing.T) {
	modes := []string{"global", "per_file", "both", "invalid_mode", ""}

	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			config := DefaultConfig()
			config.DatabaseURL = "skip"
			config.Safety.ConfidenceMode = mode

			server, err := NewStdioServer(config)
			if err != nil {
				t.Logf("Config with mode '%s' failed: %v", mode, err)
			} else {
				t.Logf("Config with mode '%s' succeeded", mode)
				server.Close()
			}
		})
	}
}

// TestConfigEdgeCases tests edge cases in configuration
func TestConfigEdgeCases(t *testing.T) {
	t.Run("empty_backup_suffix", func(t *testing.T) {
		config := DefaultConfig()
		config.DatabaseURL = "skip"
		config.Safety.BackupSuffix = ""

		server, err := NewStdioServer(config)
		if err != nil {
			t.Errorf("Empty backup suffix should be valid: %v", err)
		} else {
			server.Close()
		}
	})

	t.Run("very_long_backup_suffix", func(t *testing.T) {
		config := DefaultConfig()
		config.DatabaseURL = "skip"
		config.Safety.BackupSuffix = ".very.long.backup.suffix.that.might.cause.issues"

		server, err := NewStdioServer(config)
		if err != nil {
			t.Errorf("Long backup suffix should be valid: %v", err)
		} else {
			server.Close()
		}
	})

	t.Run("extreme_file_limits", func(t *testing.T) {
		config := DefaultConfig()
		config.DatabaseURL = "skip"
		config.Safety.MaxFiles = 1000000
		config.Safety.MaxFileSize = 1024 * 1024 * 1024       // 1GB
		config.Safety.MaxTotalSize = 10 * 1024 * 1024 * 1024 // 10GB

		server, err := NewStdioServer(config)
		if err != nil {
			t.Errorf("Extreme file limits should be valid: %v", err)
		} else {
			server.Close()
		}
	})
}
