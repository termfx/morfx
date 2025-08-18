//go:build !sqlite_fts5

package db

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOpenAndMigrateNoFTS5(t *testing.T) {
	// Ensure .morfx directory exists and clean it up before and after
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}
	defer os.RemoveAll(".morfx")
	t.Run("FTS5 disabled (fallback)", func(t *testing.T) {
		// Simulate a build without FTS5 by setting GOFLAGS
		t.Setenv("GOFLAGS", "-tags=no_fts5")

		// Clean up .morfx directory for this subtest
		_ = os.RemoveAll(".morfx")
		if err := os.MkdirAll(".morfx", 0o700); err != nil {
			t.Fatalf("Failed to create .morfx directory: %v", err)
		}

		dbConn, err := Open()
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer dbConn.Close()

		if err := Migrate(dbConn.DB); err != nil {
			t.Fatalf("Migrate failed: %v", err)
		}

		// Verify that logs table is a regular table (not FTS5)
		var tblName, tblType string
		err = dbConn.QueryRow("SELECT name, type FROM sqlite_master WHERE name = 'logs'").Scan(&tblName, &tblType)
		if err != nil {
			t.Fatalf("Failed to query sqlite_master: %v", err)
		}
		if tblType != "table" {
			t.Errorf("Expected logs table to be a regular table, got type %s", tblType)
		}

		// Check that it's not a virtual table (FTS5)
		var sqlStmt string
		err = dbConn.QueryRow("SELECT sql FROM sqlite_master WHERE name = 'logs'").Scan(&sqlStmt)
		if err != nil {
			t.Fatalf("Failed to query sqlite_master for SQL: %v", err)
		}
		if strings.Contains(sqlStmt, "VIRTUAL TABLE") {
			t.Errorf("Expected logs table not to be a VIRTUAL TABLE, but it is: %s", sqlStmt)
		}
	})
}

func TestSearchLogsNoFTS5(t *testing.T) {
	// Ensure .morfx directory exists and clean it up before and after
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}
	defer os.RemoveAll(".morfx")

	// Simulate a build without FTS5 by setting GOFLAGS
	t.Setenv("GOFLAGS", "-tags=no_fts5")

	dbConn, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer dbConn.Close()

	if err := Migrate(dbConn.DB); err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Create required parent records to satisfy foreign key constraints
	// First create a run
	runID := uuid.New().String()
	_, err = dbConn.Exec("INSERT INTO runs (id, public_ulid, status, started_at) VALUES (?, ?, ?, ?)", runID, "test-ulid", "running", 1)
	if err != nil {
		t.Fatalf("Failed to insert run: %v", err)
	}

	// Create files
	fileIDs := []string{uuid.New().String(), uuid.New().String(), uuid.New().String(), uuid.New().String()}
	for i, fileID := range fileIDs {
		_, err = dbConn.Exec("INSERT INTO files (id, run_id, path, status) VALUES (?, ?, ?, ?)", fileID, runID, fmt.Sprintf("test%d.go", i+1), "pending")
		if err != nil {
			t.Fatalf("Failed to insert file: %v", err)
		}
	}

	// Create operations
	opIDs := []string{"op1", "op2", "op3", "op4"}
	for i, opID := range opIDs {
		_, err = dbConn.Exec("INSERT INTO operations (id, run_id, file_id, seq, kind, status, started_at) VALUES (?, ?, ?, ?, ?, ?, ?)", opID, runID, fileIDs[i], i+1, "test", "running", 1)
		if err != nil {
			t.Fatalf("Failed to insert operation: %v", err)
		}
	}

	// Insert some dummy log data
	logEntries := []LogEntry{
		{OpID: "op1", TS: 1, Level: "INFO", Text: "This is a test log entry."},
		{OpID: "op2", TS: 2, Level: "WARN", Text: "Another log entry with a warning."},
		{OpID: "op3", TS: 3, Level: "ERROR", Text: "Critical error occurred."},
		{OpID: "op4", TS: 4, Level: "INFO", Text: "This log entry is about testing."},
	}

	for _, entry := range logEntries {
		_, err := dbConn.Exec("INSERT INTO logs (op_id, ts, level, text) VALUES (?, ?, ?, ?)", entry.OpID, entry.TS, entry.Level, entry.Text)
		if err != nil {
			t.Fatalf("Failed to insert log entry: %v", err)
		}
	}

	t.Run("LIKE Search (No FTS5)", func(t *testing.T) {
		results, err := SearchLogs(dbConn.DB, "test", false)
		if err != nil {
			t.Fatalf("LIKE search failed: %v", err)
		}
		if len(results) != 2 {
			t.Errorf("Expected 2 results for 'test', got %d: %+v", len(results), results)
		}
		if results[0].OpID != "op1" && results[1].OpID != "op1" {
			t.Errorf("Expected op1 in results, got %+v", results)
		}
		if results[0].OpID != "op4" && results[1].OpID != "op4" {
			t.Errorf("Expected op4 in results, got %+v", results)
		}

		results, err = SearchLogs(dbConn.DB, "warning", false)
		if err != nil {
			t.Fatalf("LIKE search failed: %v", err)
		}
		if len(results) != 1 || results[0].OpID != "op2" {
			t.Errorf("Expected 1 result for 'warning', got %d: %+v", len(results), results)
		}

		results, err = SearchLogs(dbConn.DB, "error", false)
		if err != nil {
			t.Fatalf("LIKE search failed: %v", err)
		}
		if len(results) != 1 || results[0].OpID != "op3" {
			t.Errorf("Expected 1 result for 'error', got %d: %+v", len(results), results)
		}

		results, err = SearchLogs(dbConn.DB, "nonexistent", false)
		if err != nil {
			t.Fatalf("LIKE search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for 'nonexistent', got %d: %+v", len(results), results)
		}
	})
}
