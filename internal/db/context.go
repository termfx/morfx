package db

import "github.com/termfx/morfx/internal/types"

// Context holds database configuration that was previously imported from config
type Context struct {
	DBPath            string
	ActiveKeyVersion  int
	EncryptionKeys    map[int][]byte
	KeyDerivationSalt []byte
	EncryptionMode    string
	MasterKey         string
	EncryptionAlgo    string
	RetentionRuns     int
}

// NewContext creates a new database context from configuration
func NewContext(cfg *types.DBConfig) *Context {
	return &Context{
		DBPath:            cfg.DBPath,
		ActiveKeyVersion:  cfg.ActiveKeyVersion,
		EncryptionKeys:    cfg.EncryptionKeys,
		KeyDerivationSalt: cfg.KeyDerivationSalt,
		EncryptionMode:    cfg.EncryptionMode,
		MasterKey:         cfg.MasterKey,
		EncryptionAlgo:    cfg.EncryptionAlgo,
		RetentionRuns:     cfg.RetentionRuns,
	}
}

// Global context for backward compatibility (will be set by the application)
var globalContext *Context

// SetGlobalContext sets the global database context
func SetGlobalContext(ctx *Context) {
	globalContext = ctx
}

// GetGlobalContext returns the global database context
func GetGlobalContext() *Context {
	if globalContext == nil {
		// Return a default context for backward compatibility
		return &Context{
			DBPath:           ".morfx.db",
			ActiveKeyVersion: 1,
		}
	}
	return globalContext
}
