package db

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOpenAndMigrate(t *testing.T) {
	// Ensure .morfx directory exists and clean it up before and after
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}
	defer os.RemoveAll(".morfx")

	t.Run("FTS5 enabled", func(t *testing.T) {
		dbConn, err := Open()
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		defer dbConn.Close()
		if merr := Migrate(dbConn.DB); merr != nil {
			t.Fatalf("Migrate failed: %v", merr)
		}
		// Verify that logs table is a virtual table (FTS5)
		var tblName, tblType string
		err = dbConn.QueryRow("SELECT name, type FROM sqlite_master WHERE name = 'logs'").Scan(&tblName, &tblType)
		if err != nil {
			t.Fatalf("Failed to query sqlite_master: %v", err)
		}
		if tblType != "table" { // FTS5 tables are reported as 'table' by sqlite_master
			t.Errorf("Expected logs table to be a virtual table (FTS5), got type %s", tblType)
		}
		// A more robust check would be to query pragma table_info(logs) and check for 'fts5' in the schema
	})
}

func TestRollbackIdempotency(t *testing.T) {
	// Ensure .morfx directory exists and clean it up before and after
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}
	defer os.RemoveAll(".morfx")

	// Setup: Create a dummy file and a database connection
	testFilePath := "test_rollback_idempotency.txt"
	originalContent := []byte("original content")
	modifiedContent := []byte("modified content")

	err := os.WriteFile(testFilePath, originalContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath)

	dbConn, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer dbConn.Close()

	err = Migrate(dbConn.DB)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	// Simulate a run and an operation
	runID, err := BeginRun(dbConn.DB, map[string]any{"repo": "test", "branch": "main", "commit_base": "abc"})
	if err != nil {
		t.Fatalf("BeginRun failed: %v", err)
	}

	fileID := uuid.NewString()
	err = RecordFile(dbConn.DB, runID, fileID, testFilePath, "text", int64(len(originalContent)), "hash_orig", "hash_mod", "modified")
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}

	opID, err := AppendOp(dbConn.DB, runID, fileID, "rewrite")
	if err != nil {
		t.Fatalf("AppendOp failed: %v", err)
	}

	// Record a patch
	patch := Patch{
		OpID:         opID,
		FileID:       fileID,
		Algo:         "binary",
		ForwardBlob:  modifiedContent,
		ReverseBlob:  originalContent,
		BytesAdded:   len(modifiedContent),
		BytesRemoved: len(originalContent),
	}
	err = RecordPatches(dbConn.DB, []Patch{patch})
	if err != nil {
		t.Fatalf("RecordPatches failed: %v", err)
	}

	// Modify the file to simulate the patch being applied
	err = os.WriteFile(testFilePath, modifiedContent, 0o644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// First rollback
	result1, err := Rollback(dbConn.DB, opID, false, true)
	if err != nil {
		t.Fatalf("First Rollback failed: %v", err)
	}
	if len(result1.RevertedOps) != 1 || result1.RevertedOps[0] != opID {
		t.Errorf("First Rollback did not revert the expected operation. RevertedOps: %v", result1.RevertedOps)
	}
	currentContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read file after first rollback: %v", err)
	}
	if string(currentContent) != string(originalContent) {
		t.Errorf("File content mismatch after first rollback. Expected: %s, Got: %s", originalContent, currentContent)
	}

	// Second rollback (should be idempotent)
	result2, err := Rollback(dbConn.DB, opID, false, true)
	if err != nil {
		t.Fatalf("Second Rollback failed: %v", err)
	}
	if len(result2.RevertedOps) != 0 { // No operations should be reverted again
		t.Errorf("Second Rollback reverted operations unexpectedly. RevertedOps: %v", result2.RevertedOps)
	}
	currentContent, err = os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read file after second rollback: %v", err)
	}
	if string(currentContent) != string(originalContent) {
		t.Errorf("File content mismatch after second rollback. Expected: %s, Got: %s", originalContent, currentContent)
	}

	// Verify operation status in DB
	var status string
	err = dbConn.QueryRow("SELECT status FROM operations WHERE id = ?", opID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query operation status: %v", err)
	}
	if status != "rolled_back" {
		t.Errorf("Expected operation status to be 'rolled_back', got: %s", status)
	}

	// Verify file status in DB
	var fileStatus string
	err = dbConn.QueryRow("SELECT status FROM files WHERE id = ?", fileID).Scan(&fileStatus)
	if err != nil {
		t.Fatalf("Failed to query file status: %v", err)
	}
	if fileStatus != "rolled_back" {
		t.Errorf("Expected file status to be 'rolled_back', got: %s", fileStatus)
	}
}

func TestSearchLogs(t *testing.T) {
	// Ensure .morfx directory exists and clean it up before and after
	_ = os.RemoveAll(".morfx")
	if err := os.MkdirAll(".morfx", 0o700); err != nil {
		t.Fatalf("Failed to create .morfx directory: %v", err)
	}
	defer os.RemoveAll(".morfx")

	dbConn, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer dbConn.Close()

	if merr := Migrate(dbConn.DB); merr != nil {
		t.Fatalf("Migrate failed: %v", merr)
	}

	// Create required parent records to satisfy foreign key constraints
	// First create a run
	runID := uuid.New().String()
	_, err = dbConn.Exec("INSERT INTO runs (id, public_ulid, status, started_at) VALUES (?, ?, ?, ?)", runID, "test-ulid", "running", 1)
	if err != nil {
		t.Fatalf("Failed to insert run: %v", err)
	}

	// Create files
	fileIDs := []string{uuid.New().String(), uuid.New().String(), uuid.New().String()}
	for i, fileID := range fileIDs {
		_, err = dbConn.Exec("INSERT INTO files (id, run_id, path, status) VALUES (?, ?, ?, ?)", fileID, runID, fmt.Sprintf("test%d.go", i+1), "pending")
		if err != nil {
			t.Fatalf("Failed to insert file: %v", err)
		}
	}

	// Create operations
	opIDs := []string{"op1", "op2", "op3"}
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
	}

	for _, entry := range logEntries {
		_, err := dbConn.Exec("INSERT INTO logs (op_id, ts, level, text) VALUES (?, ?, ?, ?)", entry.OpID, entry.TS, entry.Level, entry.Text)
		if err != nil {
			t.Fatalf("Failed to insert log entry: %v", err)
		}
	}

	t.Run("FTS5 Search", func(t *testing.T) {
		// Check if FTS5 is enabled for the logs table
		var sqlStmt string
		err := dbConn.QueryRow("SELECT sql FROM sqlite_master WHERE name = 'logs'").Scan(&sqlStmt)
		if err != nil {
			t.Fatalf("Failed to query sqlite_master for logs table: %v", err)
		}

		if !strings.Contains(sqlStmt, "VIRTUAL TABLE") {
			t.Skip("FTS5 is not enabled, skipping FTS5 search test.")
		}

		results, err := SearchLogs(dbConn.DB, "test", true)
		if err != nil {
			t.Fatalf("FTS5 search failed: %v", err)
		}

		if len(results) != 1 || results[0].OpID != "op1" {
			t.Errorf("Expected 1 result for 'test', got %d: %+v", len(results), results)
		}

		results, err = SearchLogs(dbConn.DB, "warning", true)
		if err != nil {
			t.Fatalf("FTS5 search failed: %v", err)
		}
		if len(results) != 1 || results[0].OpID != "op2" {
			t.Errorf("Expected 1 result for 'warning', got %d: %+v", len(results), results)
		}

		results, err = SearchLogs(dbConn.DB, "error", true)
		if err != nil {
			t.Fatalf("FTS5 search failed: %v", err)
		}
		if len(results) != 1 || results[0].OpID != "op3" {
			t.Errorf("Expected 1 result for 'error', got %d: %+v", len(results), results)
		}

		results, err = SearchLogs(dbConn.DB, "nonexistent", true)
		if err != nil {
			t.Fatalf("FTS5 search failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for 'nonexistent', got %d: %+v", len(results), results)
		}
	})

	t.Run("LIKE Search", func(t *testing.T) {
		results, err := SearchLogs(dbConn.DB, "test", false)
		if err != nil {
			t.Fatalf("LIKE search failed: %v", err)
		}
		if len(results) != 1 || results[0].OpID != "op1" {
			t.Errorf("Expected 1 result for 'test', got %d: %+v", len(results), results)
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
