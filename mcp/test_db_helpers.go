package mcp

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// setupAsyncStagingDB creates a temporary SQLite database suitable for both unit and integration tests.
func setupAsyncStagingDB(t *testing.T) *gorm.DB {
	t.Helper()
	tempDB := t.TempDir() + "/test.db"
	db, err := gorm.Open(sqlite.Open(tempDB+"?cache=shared&mode=rwc"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&models.Session{}, &models.Stage{}, &models.Apply{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
