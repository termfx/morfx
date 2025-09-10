package db

import (
	"fmt"
	"strings"
	
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/termfx/morfx/models"
)

// Connect establishes a database connection and runs migrations
func Connect(dsn string, debug bool) (*gorm.DB, error) {
	// Try to create database if it doesn't exist
	if err := ensureDatabase(dsn); err != nil && debug {
		fmt.Printf("[WARN] Could not ensure database exists: %v\n", err)
	}
	
	config := &gorm.Config{}
	
	// Enable debug logging if requested
	if debug {
		config.Logger = logger.Default.LogMode(logger.Info)
	}
	
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	
	// Run migrations
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	
	return db, nil
}

// ensureDatabase creates the database if it doesn't exist
func ensureDatabase(dsn string) error {
	// Parse database name from DSN
	dbName := extractDBName(dsn)
	if dbName == "" {
		return fmt.Errorf("could not extract database name from DSN")
	}
	
	// Connect to postgres database first
	adminDSN := strings.Replace(dsn, "/"+dbName, "/postgres", 1)
	
	db, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres db: %w", err)
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	
	// Create database if not exists
	var exists bool
	db.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", dbName).Scan(&exists)
	
	if !exists {
		if err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName)).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
	}
	
	return nil
}

// extractDBName extracts database name from DSN
func extractDBName(dsn string) string {
	// Simple extraction: postgres://user:pass@host/dbname?params
	parts := strings.Split(dsn, "/")
	if len(parts) < 4 {
		return ""
	}
	
	dbPart := parts[3]
	if idx := strings.Index(dbPart, "?"); idx > 0 {
		dbPart = dbPart[:idx]
	}
	
	return dbPart
}

// Migrate runs database migrations
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Stage{},
		&models.Apply{},
		&models.Session{},
	)
}
