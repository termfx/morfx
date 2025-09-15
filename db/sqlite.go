package db

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/termfx/morfx/models"
)

// Connect establishes a database connection and runs migrations
func Connect(dsn string, debug bool) (*gorm.DB, error) {
	// Ensure directory exists for file-based SQLite
	if !isURL(dsn) {
		dir := filepath.Dir(dsn)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	config := &gorm.Config{}

	// Enable debug logging if requested
	if debug {
		config.Logger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(sqlite.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Enable foreign keys for SQLite
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Exec("PRAGMA foreign_keys = ON")
	}

	// Run migrations
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return db, nil
}

// isURL checks if the DSN is a URL (for Turso) or file path
func isURL(dsn string) bool {
	return len(dsn) > 7 && (dsn[:7] == "http://" || dsn[:8] == "https://" || dsn[:6] == "libsql")
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Stage{},
		&models.Apply{},
		&models.Session{},
	)
}
