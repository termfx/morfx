package db

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name          string
		dsn           string
		debug         bool
		setupFunc     func(string)
		expectedError bool
		errorContains string
	}{
		{
			name:          "successful connection with memory database",
			dsn:           ":memory:",
			debug:         false,
			expectedError: false,
		},
		{
			name:          "successful connection with debug enabled",
			dsn:           ":memory:",
			debug:         true,
			expectedError: false,
		},
		{
			name:          "successful connection with file database",
			dsn:           "/tmp/test_morfx.db",
			debug:         false,
			expectedError: false,
		},
		{
			name:          "connection with nested directory creation",
			dsn:           "/tmp/nested/path/test_morfx.db",
			debug:         false,
			expectedError: false,
		},
		{
			name:          "connection with URL DSN (Turso)",
			dsn:           "libsql://127.0.0.1:19999",
			debug:         false,
			expectedError: true, // Will fail without proper credentials
			errorContains: "failed to connect",
		},
		{
			name:          "connection with HTTP URL",
			dsn:           "http://127.0.0.1:19999/db",
			debug:         false,
			expectedError: true,
			errorContains: "failed to connect",
		},
		{
			name:          "connection with HTTPS URL",
			dsn:           "https://127.0.0.1:19999/db",
			debug:         false,
			expectedError: true,
			errorContains: "failed to connect",
		},
		{
			name: "connection with invalid directory permissions",
			dsn:  "/root/restricted/test_morfx.db",
			setupFunc: func(dsn string) {
				// This will fail on most systems due to permissions
			},
			expectedError: true,
			errorContains: "failed to create database directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.setupFunc != nil {
				tt.setupFunc(tt.dsn)
			}

			// Cleanup file databases after test
			if !isURL(tt.dsn) && tt.dsn != ":memory:" {
				defer func() {
					if !tt.expectedError {
						os.Remove(tt.dsn)
						// Try to remove parent directories if they're empty
						os.Remove(filepath.Dir(tt.dsn))
					}
				}()
			}

			// Execute
			db, err := Connect(tt.dsn, tt.debug)

			// Assert
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, db)
			} else {
				require.NoError(t, err)
				require.NotNil(t, db)

				// Verify connection is working
				sqlDB, err := db.DB()
				require.NoError(t, err)
				require.NoError(t, sqlDB.Ping())

				// Verify foreign keys are enabled for SQLite
				var fkEnabled int
				err = db.Raw("PRAGMA foreign_keys").Scan(&fkEnabled).Error
				require.NoError(t, err)
				assert.Equal(t, 1, fkEnabled)

				// Verify tables were created by migration
				tables := []string{"stages", "applies", "sessions"}
				for _, table := range tables {
					assert.True(t, db.Migrator().HasTable(table), "Table %s should exist", table)
				}

				// Test basic operations
				testBasicOperations(t, db)

				// Cleanup
				sqlDB.Close()
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		dsn      string
		expected bool
	}{
		{
			name:     "HTTP URL",
			dsn:      "http://example.com",
			expected: true,
		},
		{
			name:     "HTTPS URL",
			dsn:      "https://example.com",
			expected: true,
		},
		{
			name:     "libsql URL",
			dsn:      "libsql://test.turso.io",
			expected: true,
		},
		{
			name:     "file path",
			dsn:      "/path/to/database.db",
			expected: false,
		},
		{
			name:     "relative file path",
			dsn:      "database.db",
			expected: false,
		},
		{
			name:     "memory database",
			dsn:      ":memory:",
			expected: false,
		},
		{
			name:     "empty string",
			dsn:      "",
			expected: false,
		},
		{
			name:     "short string",
			dsn:      "http",
			expected: false,
		},
		{
			name:     "almost HTTP",
			dsn:      "http:/",
			expected: false,
		},
		{
			name:     "almost HTTPS",
			dsn:      "https:/",
			expected: false,
		},
		{
			name:     "almost libsql",
			dsn:      "libsq",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isURL(tt.dsn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMigrate(t *testing.T) {
	tests := []struct {
		name          string
		setupDB       func() *gorm.DB
		expectedError bool
		errorContains string
	}{
		{
			name: "successful migration with clean database",
			setupDB: func() *gorm.DB {
				db, err := Connect(":memory:", false)
				require.NoError(t, err)
				return db
			},
			expectedError: false,
		},
		{
			name: "migration with existing tables",
			setupDB: func() *gorm.DB {
				db, err := Connect(":memory:", false)
				require.NoError(t, err)
				// Tables are already created by Connect(), so this tests re-migration
				return db
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := tt.setupDB()
			defer func() {
				sqlDB, _ := db.DB()
				if sqlDB != nil {
					sqlDB.Close()
				}
			}()

			// Drop tables to test fresh migration
			if tt.name == "successful migration with clean database" {
				db.Migrator().DropTable(&models.Apply{})
				db.Migrator().DropTable(&models.Stage{})
				db.Migrator().DropTable(&models.Session{})
			}

			err := Migrate(db)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)

				// Verify all tables exist
				assert.True(t, db.Migrator().HasTable(&models.Stage{}))
				assert.True(t, db.Migrator().HasTable(&models.Apply{}))
				assert.True(t, db.Migrator().HasTable(&models.Session{}))

				// Verify table structure by creating sample records
				testBasicOperations(t, db)
			}
		})
	}
}

func TestConnectWithInvalidPath(t *testing.T) {
	// Test with a path that contains invalid characters (on some systems)
	invalidPaths := []string{
		"/dev/null/cannot_create_here.db",
	}

	for _, path := range invalidPaths {
		t.Run(fmt.Sprintf("invalid_path_%s", filepath.Base(path)), func(t *testing.T) {
			db, err := Connect(path, false)

			// This should either fail with directory creation error or connection error
			if err != nil {
				assert.Contains(t, err.Error(), "failed to")
				assert.Nil(t, db)
			} else {
				// If it somehow succeeds, clean up
				if db != nil {
					sqlDB, _ := db.DB()
					if sqlDB != nil {
						sqlDB.Close()
					}
				}
			}
		})
	}
}

func TestConnectDebugMode(t *testing.T) {
	// Test debug mode configuration
	db, err := Connect(":memory:", true)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Verify the database is functional
	assert.True(t, db.Migrator().HasTable(&models.Stage{}))
}

// testBasicOperations performs basic CRUD operations to verify database functionality
func testBasicOperations(t *testing.T, db *gorm.DB) {
	// Test Session creation
	session := &models.Session{
		ID: "test-session-123",
	}
	err := db.Create(session).Error
	assert.NoError(t, err)

	// Test Stage creation
	stage := &models.Stage{
		ID:        "test-stage-123",
		SessionID: session.ID,
		Language:  "go",
		Operation: "replace",
		Status:    "pending",
	}
	err = db.Create(stage).Error
	assert.NoError(t, err)

	// Test Apply creation
	apply := &models.Apply{
		ID:      "test-apply-123",
		StageID: stage.ID,
	}
	err = db.Create(apply).Error
	assert.NoError(t, err)

	// Test reading back
	var retrievedStage models.Stage
	err = db.Where("id = ?", stage.ID).First(&retrievedStage).Error
	assert.NoError(t, err)
	assert.Equal(t, stage.Language, retrievedStage.Language)

	// Test relationships
	var stageWithApply models.Stage
	err = db.Preload("Apply").Where("id = ?", stage.ID).First(&stageWithApply).Error
	assert.NoError(t, err)
	assert.NotNil(t, stageWithApply.Apply)
	assert.Equal(t, apply.ID, stageWithApply.Apply.ID)
}

func TestConnectDirectoryCreation(t *testing.T) {
	// Test nested directory creation
	tempDir := "/tmp/morfx_test_" + fmt.Sprintf("%d", os.Getpid())
	dbPath := filepath.Join(tempDir, "nested", "deep", "test.db")

	defer func() {
		os.RemoveAll(tempDir)
	}()

	db, err := Connect(dbPath, false)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Verify directory was created
	assert.DirExists(t, filepath.Dir(dbPath))

	// Verify database file exists and is functional
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestConnectForeignKeysEnabled(t *testing.T) {
	db, err := Connect(":memory:", false)
	require.NoError(t, err)
	require.NotNil(t, db)

	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.Raw("PRAGMA foreign_keys").Scan(&fkEnabled).Error
	require.NoError(t, err)
	assert.Equal(t, 1, fkEnabled, "Foreign keys should be enabled")

	// Test foreign key constraint
	session := &models.Session{ID: "test-session"}
	err = db.Create(session).Error
	require.NoError(t, err)

	// Create a stage with valid session ID
	stage := &models.Stage{
		ID:        "test-stage",
		SessionID: session.ID,
		Language:  "go",
		Operation: "test",
	}
	err = db.Create(stage).Error
	assert.NoError(t, err)

	// Try to create an apply record with non-existent stage ID
	invalidApply := &models.Apply{
		ID:      "test-apply",
		StageID: "non-existent-stage",
	}
	err = db.Create(invalidApply).Error
	assert.Error(t, err, "Should fail due to foreign key constraint")
}
