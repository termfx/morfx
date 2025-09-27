//go:build integration
// +build integration

package db

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestConnectLibSQLIntegration exercises the libSQL/Turso connection path when
// a remote DSN and auth token are available in the environment. The test is
// gated behind the "integration" build tag and skips automatically when the
// required variables are not present.
func TestConnectLibSQLIntegration(t *testing.T) {
	_ = godotenv.Load()

	dsn := os.Getenv("MORFX_DATABASE_URL")
	token := os.Getenv("MORFX_LIBSQL_AUTH_TOKEN")

	if dsn == "" || token == "" {
		t.Skip("MORFX_DATABASE_URL or MORFX_LIBSQL_AUTH_TOKEN not set; skipping")
	}

	db, err := Connect(dsn, false)
	if err != nil {
		t.Fatalf("failed to connect to remote libSQL instance: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to obtain sql.DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("ping to libSQL instance failed: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close sql.DB: %v", err)
	}
}
