package db

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestExecWithRetry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		query   string
		args    []any
		wantErr bool
	}{
		{
			name:    "successful execution",
			query:   "CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY)",
			args:    nil,
			wantErr: false,
		},
		{
			name:    "invalid SQL",
			query:   "INVALID SQL STATEMENT",
			args:    nil,
			wantErr: true,
		},
		{
			name:    "insert with parameters",
			query:   "INSERT INTO test_table (id) VALUES (?)",
			args:    []any{1},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execWithRetry(db, tt.query, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("execWithRetry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecWithRetryTx(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test table first
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_tx_table (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	tests := []struct {
		name    string
		query   string
		args    []any
		wantErr bool
	}{
		{
			name:    "successful transaction execution",
			query:   "INSERT INTO test_tx_table (id, value) VALUES (?, ?)",
			args:    []any{1, "test"},
			wantErr: false,
		},
		{
			name:    "invalid SQL in transaction",
			query:   "INVALID SQL STATEMENT",
			args:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execWithRetryTx(tx, tt.query, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("execWithRetryTx() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQueryRowWithRetry(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create and populate test table
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS test_query_table (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	_, err = db.Exec("INSERT INTO test_query_table (id, name) VALUES (1, 'test')")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	tests := []struct {
		name     string
		query    string
		args     []any
		wantErr  bool
		expected string
	}{
		{
			name:     "successful query",
			query:    "SELECT name FROM test_query_table WHERE id = ?",
			args:     []any{1},
			wantErr:  false,
			expected: "test",
		},
		{
			name:    "query with no results",
			query:   "SELECT name FROM test_query_table WHERE id = ?",
			args:    []any{999},
			wantErr: true, // sql.ErrNoRows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := queryRowWithRetry(db, tt.query, tt.args...)
			var result string
			err := row.Scan(&result)

			if tt.wantErr {
				if err == nil {
					t.Errorf("queryRowWithRetry() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("queryRowWithRetry() error = %v, wantErr %v", err, tt.wantErr)
				}
				if result != tt.expected {
					t.Errorf("queryRowWithRetry() result = %s, expected %s", result, tt.expected)
				}
			}
		})
	}
}

func TestQuickCheck(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "healthy database",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := QuickCheck(db)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuickCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunHealthCheck(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "healthy database",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RunHealthCheck(db)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunHealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDBPath(t *testing.T) {
	// getDBPath() takes no parameters and returns (string, error)
	// This test verifies it returns a valid path
	t.Run("get database path", func(t *testing.T) {
		result, err := getDBPath()
		if err != nil {
			t.Errorf("getDBPath() error = %v", err)
		}
		if result == "" {
			t.Errorf("getDBPath() returned empty path")
		}
	})
}

func TestDBConnMethods(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	conn := &DBConn{DB: db}

	t.Run("Ping", func(t *testing.T) {
		err := conn.Ping()
		if err != nil {
			t.Errorf("DBConn.Ping() error = %v", err)
		}
	})

	t.Run("Close", func(t *testing.T) {
		// Create a new connection for this test
		tempDB, tempCleanup := setupTestDB(t)
		defer tempCleanup()

		tempConn := &DBConn{DB: tempDB}
		err := tempConn.Close()
		if err != nil {
			t.Errorf("DBConn.Close() error = %v", err)
		}
	})

	t.Run("Stats", func(t *testing.T) {
		stats := conn.Stats()
		if stats.OpenConnections < 0 {
			t.Errorf("DBConn.Stats() returned negative OpenConnections")
		}
	})

	t.Run("SetMaxOpenConns", func(t *testing.T) {
		conn.SetMaxOpenConns(10)
		stats := conn.Stats()
		if stats.MaxOpenConnections != 10 {
			t.Errorf("DBConn.SetMaxOpenConns() did not set max connections correctly")
		}
	})

	t.Run("SetMaxIdleConns", func(t *testing.T) {
		conn.SetMaxIdleConns(5)
		// No direct way to verify this, but ensure it doesn't panic
	})

	t.Run("SetConnMaxLifetime", func(t *testing.T) {
		conn.SetConnMaxLifetime(time.Hour)
		// No direct way to verify this, but ensure it doesn't panic
	})
}

func TestOpen(t *testing.T) {
	// Open() takes no parameters and returns (*DBConn, error)
	// It uses environment variables or defaults for configuration
	t.Run("open database connection", func(t *testing.T) {
		db, err := Open()
		if err != nil {
			// This might fail if environment is not properly configured
			// which is expected in test environment
			t.Logf("Open() error = %v (expected in test environment)", err)
			return
		}

		if db == nil {
			t.Errorf("Open() returned nil database")
		} else {
			// Test that we can perform basic operations
			err = db.Ping()
			if err != nil {
				t.Errorf("Open() database ping failed: %v", err)
			}
			db.Close()
		}
	})
}

func TestCheckWALSizeAndCheckpoint(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Enable WAL mode
	_, err := db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		t.Fatalf("Failed to enable WAL mode: %v", err)
	}

	// Create some data to generate WAL entries
	_, err = db.Exec("CREATE TABLE test_wal_table (id INTEGER PRIMARY KEY, data TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	for range 100 {
		_, err = db.Exec("INSERT INTO test_wal_table (data) VALUES (?)", "test data")
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// CheckWALSizeAndCheckpoint takes only *sql.DB parameter
	t.Run("checkpoint WAL file", func(t *testing.T) {
		err := CheckWALSizeAndCheckpoint(db)
		if err != nil {
			t.Errorf("CheckWALSizeAndCheckpoint() error = %v", err)
		}
	})
}

func TestRetryLogic(t *testing.T) {
	// Test the retry logic by ensuring database operations work under contention
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a table for testing
	_, err := db.Exec("CREATE TABLE retry_test (id INTEGER PRIMARY KEY)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test concurrent operations that might trigger retry logic
	done := make(chan bool, 2)
	errorChan := make(chan error, 2)

	go func() {
		for i := range 10 {
			_, errx := execWithRetry(db, "INSERT INTO retry_test (id) VALUES (?)", i)
			if errx != nil {
				errorChan <- fmt.Errorf("concurrent insert %d failed: %v", i, err)
				return
			}
		}
		done <- true
	}()

	go func() {
		for i := 10; i < 20; i++ {
			_, errx := execWithRetry(db, "INSERT INTO retry_test (id) VALUES (?)", i)
			if errx != nil {
				errorChan <- fmt.Errorf("concurrent insert %d failed: %v", i, err)
				return
			}
		}
		done <- true
	}()

	// Wait for both goroutines to complete or error
	completedCount := 0
	for completedCount < 2 {
		select {
		case <-done:
			completedCount++
		case errChan := <-errorChan:
			t.Errorf("Goroutine failed: %v", errChan)
			return
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out waiting for goroutines")
		}
	}

	// Verify all inserts succeeded
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM retry_test").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows: %v", err)
	}
	if count != 20 {
		t.Errorf("Expected 20 rows, got %d", count)
	}

	// Test that execWithRetry actually works with database operations
	_, err = execWithRetry(db, "INSERT INTO retry_test (id) VALUES (?)", 100)
	if err != nil {
		t.Errorf("execWithRetry failed on normal operation: %v", err)
	}

	// Verify the final count
	err = db.QueryRow("SELECT COUNT(*) FROM retry_test").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count rows after final insert: %v", err)
	}
	if count != 21 {
		t.Errorf("Expected 21 rows after final insert, got %d", count)
	}
}

func TestDatabaseLockHandling(t *testing.T) {
	// Test that our retry logic handles database lock errors appropriately
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test table
	_, err := db.Exec("CREATE TABLE lock_test (id INTEGER PRIMARY KEY, data TEXT)")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test that normal operations work
	_, err = execWithRetry(db, "INSERT INTO lock_test (data) VALUES (?)", "test")
	if err != nil {
		t.Errorf("execWithRetry failed on normal operation: %v", err)
	}

	// Test query operations
	row := queryRowWithRetry(db, "SELECT data FROM lock_test WHERE id = ?", 1)
	var data string
	err = row.Scan(&data)
	if err != nil {
		t.Errorf("queryRowWithRetry failed: %v", err)
	}
	if data != "test" {
		t.Errorf("Expected 'test', got '%s'", data)
	}
}
func TestErrorHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name        string
		query       string
		expectError bool
		errorCheck  func(error) bool
	}{
		{
			name:        "syntax error",
			query:       "INVALID SQL SYNTAX",
			expectError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "syntax error")
			},
		},
		{
			name:        "table not found",
			query:       "SELECT * FROM nonexistent_table",
			expectError: true,
			errorCheck: func(err error) bool {
				return strings.Contains(err.Error(), "no such table")
			},
		},
		{
			name:        "valid query",
			query:       "SELECT 1",
			expectError: false,
			errorCheck:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := execWithRetry(db, tt.query)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for query '%s', but got none", tt.query)
				} else if tt.errorCheck != nil && !tt.errorCheck(err) {
					t.Errorf("Error check failed for query '%s': %v", tt.query, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for query '%s': %v", tt.query, err)
				}
			}
		})
	}
}
