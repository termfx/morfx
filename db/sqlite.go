package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	libsql "github.com/tursodatabase/libsql-client-go/libsql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/oxhq/morfx/internal/securefs"
	"github.com/oxhq/morfx/models"
)

// Connect establishes a database connection and runs migrations
func Connect(dsn string, debug bool) (*gorm.DB, error) {
	// Ensure directory exists for file-based SQLite
	if !isURL(dsn) {
		dir := filepath.Dir(dsn)
		if err := securefs.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	config := &gorm.Config{}

	// Configure GORM logger: always write to stderr (never stdout)
	// to avoid contaminating MCP JSON-RPC stdio transport
	if debug {
		config.Logger = logger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	} else {
		config.Logger = logger.Default.LogMode(logger.Silent)
	}

	var (
		dialector gorm.Dialector
		conn      *sql.DB
	)
	if isURL(dsn) {
		var (
			connector driver.Connector
			err       error
		)

		token := os.Getenv("MORFX_LIBSQL_AUTH_TOKEN")
		if token != "" {
			connector, err = libsql.NewConnector(dsn, libsql.WithAuthToken(token))
		} else {
			connector, err = libsql.NewConnector(dsn)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create libsql connector: %w", err)
		}

		conn = sql.OpenDB(connector)
		dialector = sqlite.Dialector{
			DriverName: "libsql",
			Conn:       conn,
			DSN:        dsn,
		}
	} else {
		dialector = sqlite.Open(dsn)
	}

	db, err := gorm.Open(dialector, config)
	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Enable foreign keys for SQLite
	if sqlDB, err := db.DB(); err == nil {
		if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
		}
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
