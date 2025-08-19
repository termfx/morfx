package db

import (
	"os"
	"testing"

	"github.com/termfx/morfx/internal/types"
)

func TestMain(m *testing.M) {
	// Set up the global context for the tests
	cfg := &types.DBConfig{
		MasterKey:      os.Getenv("MORFX_MASTER_KEY"),
		EncryptionMode: os.Getenv("MORFX_ENCRYPTION_MODE"),
		EncryptionAlgo: os.Getenv("MORFX_ENCRYPTION_ALGO"),
	}
	// Set defaults if environment variables are empty
	if cfg.EncryptionMode == "" {
		cfg.EncryptionMode = "auto"
	}
	if cfg.EncryptionAlgo == "" {
		cfg.EncryptionAlgo = "xchacha20poly1305"
	}
	ctx := NewContext(cfg)
	SetGlobalContext(ctx)

	// Run the tests
	os.Exit(m.Run())
}
