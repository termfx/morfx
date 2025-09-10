package mcp

import "time"

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
	
	// Debug
	Debug bool
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() Config {
	return Config{
		DatabaseURL:          "postgres://root:@localhost/morfx_dev?sslmode=disable",
		AutoApplyEnabled:     true,
		AutoApplyThreshold:   0.85,
		StagingTTL:          15 * time.Minute,
		MaxStagesPerSession:  100,
		MaxAppliesPerSession: 10,
		Debug:               false,
	}
}
