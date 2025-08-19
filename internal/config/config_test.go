package config

import (
	"os"
	"testing"
)

func TestLoadConfig_DefaultValues(t *testing.T) {
	// Clear all environment variables
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	cfg := LoadConfig()

	// Test default values
	if cfg.EncryptionMode != "auto" {
		t.Errorf("Expected EncryptionMode 'auto', got '%s'", cfg.EncryptionMode)
	}
	if cfg.EncryptionAlgo != "xchacha20poly1305" {
		t.Errorf("Expected EncryptionAlgo 'xchacha20poly1305', got '%s'", cfg.EncryptionAlgo)
	}
	if cfg.ActiveKeyVersion != 0 {
		t.Errorf("Expected ActiveKeyVersion 0, got %d", cfg.ActiveKeyVersion)
	}
	if cfg.WALAutoCheckpointMB != 128 {
		t.Errorf("Expected WALAutoCheckpointMB 128, got %d", cfg.WALAutoCheckpointMB)
	}
	if cfg.RetentionRuns != 20 {
		t.Errorf("Expected RetentionRuns 20, got %d", cfg.RetentionRuns)
	}
	if cfg.MasterKey != "" {
		t.Errorf("Expected empty MasterKey, got '%s'", cfg.MasterKey)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	// Set environment variables
	os.Setenv("MORFX_ENCRYPTION_MODE", "blob")
	os.Setenv("MORFX_MASTER_KEY", "test-key-123")
	os.Setenv("MORFX_ENCRYPTION_ALGO", "aes256")
	os.Setenv("MORFX_ACTIVE_KEY_VERSION", "5")
	os.Setenv("MORFX_DB_WAL_AUTOCHECKPOINT_MB", "256")
	os.Setenv("MORFX_DB_RETENTION_RUNS", "50")

	cfg := LoadConfig()

	// Test environment variable values
	if cfg.EncryptionMode != "blob" {
		t.Errorf("Expected EncryptionMode 'blob', got '%s'", cfg.EncryptionMode)
	}
	if cfg.MasterKey != "test-key-123" {
		t.Errorf("Expected MasterKey 'test-key-123', got '%s'", cfg.MasterKey)
	}
	if cfg.EncryptionAlgo != "aes256" {
		t.Errorf("Expected EncryptionAlgo 'aes256', got '%s'", cfg.EncryptionAlgo)
	}
	if cfg.ActiveKeyVersion != 5 {
		t.Errorf("Expected ActiveKeyVersion 5, got %d", cfg.ActiveKeyVersion)
	}
	if cfg.WALAutoCheckpointMB != 256 {
		t.Errorf("Expected WALAutoCheckpointMB 256, got %d", cfg.WALAutoCheckpointMB)
	}
	if cfg.RetentionRuns != 50 {
		t.Errorf("Expected RetentionRuns 50, got %d", cfg.RetentionRuns)
	}
}

func TestLoadConfig_InvalidIntegerValues(t *testing.T) {
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	// Set invalid integer values
	os.Setenv("MORFX_ACTIVE_KEY_VERSION", "invalid")
	os.Setenv("MORFX_DB_WAL_AUTOCHECKPOINT_MB", "not-a-number")
	os.Setenv("MORFX_DB_RETENTION_RUNS", "abc")

	cfg := LoadConfig()

	// Should fall back to default values
	if cfg.ActiveKeyVersion != 0 {
		t.Errorf("Expected ActiveKeyVersion 0 (default), got %d", cfg.ActiveKeyVersion)
	}
	if cfg.WALAutoCheckpointMB != 128 {
		t.Errorf("Expected WALAutoCheckpointMB 128 (default), got %d", cfg.WALAutoCheckpointMB)
	}
	if cfg.RetentionRuns != 20 {
		t.Errorf("Expected RetentionRuns 20 (default), got %d", cfg.RetentionRuns)
	}
}

func TestLoadConfig_NegativeValues(t *testing.T) {
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	// Set negative values
	os.Setenv("MORFX_ACTIVE_KEY_VERSION", "-1")
	os.Setenv("MORFX_DB_WAL_AUTOCHECKPOINT_MB", "-10")
	os.Setenv("MORFX_DB_RETENTION_RUNS", "-5")

	cfg := LoadConfig()

	// ActiveKeyVersion should fall back to default for negative values
	if cfg.ActiveKeyVersion != 0 {
		t.Errorf("Expected ActiveKeyVersion 0 (default for negative), got %d", cfg.ActiveKeyVersion)
	}
	// WALAutoCheckpointMB should fall back to default for non-positive values
	if cfg.WALAutoCheckpointMB != 128 {
		t.Errorf("Expected WALAutoCheckpointMB 128 (default for negative), got %d", cfg.WALAutoCheckpointMB)
	}
	// RetentionRuns should fall back to default for negative values
	if cfg.RetentionRuns != 20 {
		t.Errorf("Expected RetentionRuns 20 (default for negative), got %d", cfg.RetentionRuns)
	}
}

func TestLoadConfig_ZeroValues(t *testing.T) {
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	// Set zero values
	os.Setenv("MORFX_ACTIVE_KEY_VERSION", "0")
	os.Setenv("MORFX_DB_WAL_AUTOCHECKPOINT_MB", "0")
	os.Setenv("MORFX_DB_RETENTION_RUNS", "0")

	cfg := LoadConfig()

	// ActiveKeyVersion 0 should be accepted
	if cfg.ActiveKeyVersion != 0 {
		t.Errorf("Expected ActiveKeyVersion 0, got %d", cfg.ActiveKeyVersion)
	}
	// WALAutoCheckpointMB 0 should fall back to default (not positive)
	if cfg.WALAutoCheckpointMB != 128 {
		t.Errorf("Expected WALAutoCheckpointMB 128 (default for zero), got %d", cfg.WALAutoCheckpointMB)
	}
	// RetentionRuns 0 should be accepted (>= 0)
	if cfg.RetentionRuns != 0 {
		t.Errorf("Expected RetentionRuns 0, got %d", cfg.RetentionRuns)
	}
}

func TestLoadConfig_EmptyStringValues(t *testing.T) {
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	// Set empty string values
	os.Setenv("MORFX_ENCRYPTION_MODE", "")
	os.Setenv("MORFX_ENCRYPTION_ALGO", "")
	os.Setenv("MORFX_MASTER_KEY", "")

	cfg := LoadConfig()

	// Should fall back to defaults for empty strings
	if cfg.EncryptionMode != "auto" {
		t.Errorf("Expected EncryptionMode 'auto' (default for empty), got '%s'", cfg.EncryptionMode)
	}
	if cfg.EncryptionAlgo != "xchacha20poly1305" {
		t.Errorf("Expected EncryptionAlgo 'xchacha20poly1305' (default for empty), got '%s'", cfg.EncryptionAlgo)
	}
	if cfg.MasterKey != "" {
		t.Errorf("Expected empty MasterKey, got '%s'", cfg.MasterKey)
	}
}

func TestLoadConfig_LargeValues(t *testing.T) {
	clearConfigEnvVars()
	defer clearConfigEnvVars()

	// Set large values
	os.Setenv("MORFX_ACTIVE_KEY_VERSION", "999999")
	os.Setenv("MORFX_DB_WAL_AUTOCHECKPOINT_MB", "10000")
	os.Setenv("MORFX_DB_RETENTION_RUNS", "1000000")

	cfg := LoadConfig()

	// Should accept large valid values
	if cfg.ActiveKeyVersion != 999999 {
		t.Errorf("Expected ActiveKeyVersion 999999, got %d", cfg.ActiveKeyVersion)
	}
	if cfg.WALAutoCheckpointMB != 10000 {
		t.Errorf("Expected WALAutoCheckpointMB 10000, got %d", cfg.WALAutoCheckpointMB)
	}
	if cfg.RetentionRuns != 1000000 {
		t.Errorf("Expected RetentionRuns 1000000, got %d", cfg.RetentionRuns)
	}
}

// Helper function to clear all config-related environment variables
func clearConfigEnvVars() {
	envVars := []string{
		"MORFX_ENCRYPTION_MODE",
		"MORFX_MASTER_KEY",
		"MORFX_ENCRYPTION_ALGO",
		"MORFX_ACTIVE_KEY_VERSION",
		"MORFX_DB_WAL_AUTOCHECKPOINT_MB",
		"MORFX_DB_RETENTION_RUNS",
	}
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}
