//go:build integration
// +build integration

package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTransactionLog(t *testing.T) {
	// Clean up any existing transaction logs
	os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	if tl == nil {
		t.Fatal("TransactionLog should not be nil")
	}

	if tl.transactions == nil {
		t.Error("Transactions map should be initialized")
	}

	if tl.logFile == "" {
		t.Error("Log file path should be set")
	}

	// Check that the log directory was created
	if _, err := os.Stat("./.morfx/transactions"); os.IsNotExist(err) {
		t.Error("Transaction log directory should be created")
	}

	// Check log file path format - accept both relative and absolute paths
	if !strings.Contains(tl.logFile, ".morfx/transactions/tx_") {
		t.Errorf("Log file path should contain .morfx/transactions/tx_, got %s", tl.logFile)
	}

	if !strings.HasSuffix(tl.logFile, ".log") {
		t.Errorf("Log file should have .log extension, got %s", tl.logFile)
	}

	// Cleanup
	os.RemoveAll("./.morfx")
}

func TestBeginTransaction(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	targetPath := "/test/target.txt"
	tmpPath := "/test/target.txt.tmp"
	backupPath := "/test/target.txt.backup"

	txID := tl.BeginTransaction(targetPath, tmpPath, backupPath)

	if txID == "" {
		t.Error("Transaction ID should not be empty")
	}

	// Check transaction was stored
	tl.mutex.RLock()
	tx, exists := tl.transactions[txID]
	tl.mutex.RUnlock()

	if !exists {
		t.Error("Transaction should be stored")
	}

	if tx == nil {
		t.Fatal("Transaction should not be nil")
	}

	if tx.ID != txID {
		t.Errorf("Transaction ID mismatch: expected %s, got %s", txID, tx.ID)
	}

	if tx.TargetPath != targetPath {
		t.Errorf("Target path mismatch: expected %s, got %s", targetPath, tx.TargetPath)
	}

	if tx.TmpPath != tmpPath {
		t.Errorf("Tmp path mismatch: expected %s, got %s", tmpPath, tx.TmpPath)
	}

	if tx.BackupPath != backupPath {
		t.Errorf("Backup path mismatch: expected %s, got %s", backupPath, tx.BackupPath)
	}

	if tx.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", tx.Status)
	}

	if tx.StartTime.IsZero() {
		t.Error("Start time should be set")
	}
}

func TestCompleteTransaction(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	txID := tl.BeginTransaction("/test/target.txt", "/test/target.txt.tmp", "/test/target.txt.backup")

	// Complete the transaction
	tl.CompleteTransaction(txID)

	// Check transaction status
	tl.mutex.RLock()
	tx, exists := tl.transactions[txID]
	tl.mutex.RUnlock()

	if !exists {
		t.Error("Transaction should still exist after completion")
	}

	if tx.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", tx.Status)
	}

	if tx.EndTime.IsZero() {
		t.Error("End time should be set")
	}

	// Wait a moment for cleanup goroutine to be scheduled
	time.Sleep(20 * time.Millisecond)

	// Transaction should be cleaned up after the delay (but we won't wait 5 minutes in test)
	// Just verify the cleanup function is triggered properly
}

func TestCompleteTransaction_NonExistent(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	// Complete a non-existent transaction (should not panic)
	tl.CompleteTransaction("nonexistent-tx")

	// Should not crash
}

func TestFailTransaction(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	// Create test files
	testDir := "./test_tx_fail"
	os.MkdirAll(testDir, 0o755)
	defer os.RemoveAll(testDir)

	targetPath := filepath.Join(testDir, "target.txt")
	backupPath := filepath.Join(testDir, "target.txt.backup")

	// Create original file
	originalContent := "original content"
	if err := os.WriteFile(targetPath, []byte(originalContent), 0o644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create backup
	backupContent := "backup content"
	if err := os.WriteFile(backupPath, []byte(backupContent), 0o644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	tl := NewTransactionLog()
	txID := tl.BeginTransaction(targetPath, "", backupPath)

	// Fail the transaction
	testError := fmt.Errorf("test failure")
	err := tl.FailTransaction(txID, testError)
	if err != nil {
		t.Errorf("FailTransaction should not return error: %v", err)
	}

	// Check transaction status
	tl.mutex.RLock()
	tx, exists := tl.transactions[txID]
	tl.mutex.RUnlock()

	if !exists {
		t.Error("Transaction should exist after failure")
	}

	if tx.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", tx.Status)
	}

	if tx.Error != testError.Error() {
		t.Errorf("Expected error message '%s', got '%s'", testError.Error(), tx.Error)
	}

	// Check that backup was restored
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(content) != backupContent {
		t.Errorf("Expected restored content '%s', got '%s'", backupContent, string(content))
	}
}

func TestFailTransaction_NonExistent(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	err := tl.FailTransaction("nonexistent-tx", fmt.Errorf("test error"))

	if err == nil {
		t.Error("Expected error for non-existent transaction")
	}

	if !strings.Contains(err.Error(), "transaction not found") {
		t.Errorf("Expected 'transaction not found' error, got: %v", err)
	}
}

func TestRollbackTransaction(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	// Create test files
	testDir := "./test_tx_rollback"
	os.MkdirAll(testDir, 0o755)
	defer os.RemoveAll(testDir)

	targetPath := filepath.Join(testDir, "target.txt")
	backupPath := filepath.Join(testDir, "target.txt.backup")

	// Create backup
	backupContent := "backup content for rollback"
	if err := os.WriteFile(backupPath, []byte(backupContent), 0o644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	tl := NewTransactionLog()
	txID := tl.BeginTransaction(targetPath, "", backupPath)

	// Rollback the transaction
	err := tl.RollbackTransaction(txID)
	if err != nil {
		t.Errorf("RollbackTransaction should not return error: %v", err)
	}

	// Check transaction status
	tl.mutex.RLock()
	tx, exists := tl.transactions[txID]
	tl.mutex.RUnlock()

	if !exists {
		t.Error("Transaction should exist after rollback")
	}

	if tx.Status != "rolled_back" {
		t.Errorf("Expected status 'rolled_back', got %s", tx.Status)
	}

	// Check that backup was restored
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(content) != backupContent {
		t.Errorf("Expected restored content '%s', got '%s'", backupContent, string(content))
	}
}

func TestRollbackTransaction_NonExistent(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	err := tl.RollbackTransaction("nonexistent-tx")

	if err == nil {
		t.Error("Expected error for non-existent transaction")
	}

	if !strings.Contains(err.Error(), "transaction not found") {
		t.Errorf("Expected 'transaction not found' error, got: %v", err)
	}
}

func TestRollbackTransaction_NoBackup(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	// Create test files
	testDir := "./test_tx_no_backup"
	os.MkdirAll(testDir, 0o755)
	defer os.RemoveAll(testDir)

	targetPath := filepath.Join(testDir, "target.txt")

	// Create target file
	if err := os.WriteFile(targetPath, []byte("target content"), 0o644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	tl := NewTransactionLog()
	txID := tl.BeginTransaction(targetPath, "", "") // No backup path

	// Rollback the transaction
	err := tl.RollbackTransaction(txID)
	if err != nil {
		t.Errorf("RollbackTransaction should not return error: %v", err)
	}

	// Target file should be removed since no backup exists
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("Target file should be removed when no backup exists")
	}
}

func TestGetPendingTransactions(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	// Start with no pending transactions
	pending := tl.GetPendingTransactions()
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending transactions, got %d", len(pending))
	}

	// Create some transactions
	tx1 := tl.BeginTransaction("/test/file1.txt", "", "")
	tx2 := tl.BeginTransaction("/test/file2.txt", "", "")
	tx3 := tl.BeginTransaction("/test/file3.txt", "", "")

	// Complete one transaction
	tl.CompleteTransaction(tx2)

	// Get pending transactions
	pending = tl.GetPendingTransactions()

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending transactions, got %d", len(pending))
	}

	// Check that the right transactions are pending
	pendingIDs := make(map[string]bool)
	for _, tx := range pending {
		pendingIDs[tx.ID] = true
	}

	if !pendingIDs[tx1] {
		t.Error("Transaction 1 should be pending")
	}

	if pendingIDs[tx2] {
		t.Error("Transaction 2 should not be pending (completed)")
	}

	if !pendingIDs[tx3] {
		t.Error("Transaction 3 should be pending")
	}
}

func TestRollbackAll(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	// Create test files
	testDir := "./test_rollback_all"
	os.MkdirAll(testDir, 0o755)
	defer os.RemoveAll(testDir)

	tl := NewTransactionLog()

	// Create multiple transactions
	numTx := 3
	txIDs := make([]string, numTx)
	for i := range numTx {
		targetPath := filepath.Join(testDir, fmt.Sprintf("file%d.txt", i))
		backupPath := filepath.Join(testDir, fmt.Sprintf("file%d.txt.backup", i))

		// Create backup file
		if err := os.WriteFile(backupPath, fmt.Appendf(nil, "backup %d", i), 0o644); err != nil {
			t.Fatalf("Failed to create backup file %d: %v", i, err)
		}

		txIDs[i] = tl.BeginTransaction(targetPath, "", backupPath)
	}

	// Complete one transaction (should not be rolled back)
	tl.CompleteTransaction(txIDs[1])

	// Rollback all pending transactions
	err := tl.RollbackAll()
	if err != nil {
		t.Errorf("RollbackAll should not return error: %v", err)
	}

	// Check transaction statuses
	for i, txID := range txIDs {
		tl.mutex.RLock()
		tx, exists := tl.transactions[txID]
		tl.mutex.RUnlock()

		if !exists {
			t.Errorf("Transaction %d should exist", i)
			continue
		}

		if i == 1 {
			// This transaction was completed, should not be rolled back
			if tx.Status != "completed" {
				t.Errorf("Transaction %d should remain completed, got %s", i, tx.Status)
			}
		} else {
			// Other transactions should be rolled back
			if tx.Status != "rolled_back" {
				t.Errorf("Transaction %d should be rolled back, got %s", i, tx.Status)
			}
		}
	}
}

func TestGetSummary(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	// Start with empty summary
	summary := tl.GetSummary()
	expected := TransactionSummary{
		TotalTransactions:      0,
		PendingTransactions:    0,
		CompletedTransactions:  0,
		FailedTransactions:     0,
		RolledBackTransactions: 0,
	}

	if summary != expected {
		t.Errorf("Expected empty summary %+v, got %+v", expected, summary)
	}

	// Create transactions with different statuses
	tx1 := tl.BeginTransaction("/test/file1.txt", "", "")
	tx2 := tl.BeginTransaction("/test/file2.txt", "", "")
	tx3 := tl.BeginTransaction("/test/file3.txt", "", "")
	_ = tl.BeginTransaction("/test/file4.txt", "", "")

	// Complete some transactions
	tl.CompleteTransaction(tx1)
	tl.FailTransaction(tx2, fmt.Errorf("test error"))
	tl.RollbackTransaction(tx3)
	// tx4 remains pending

	summary = tl.GetSummary()
	expected = TransactionSummary{
		TotalTransactions:      4,
		PendingTransactions:    1,
		CompletedTransactions:  1,
		FailedTransactions:     1,
		RolledBackTransactions: 1,
	}

	if summary != expected {
		t.Errorf("Expected summary %+v, got %+v", expected, summary)
	}
}

func TestGenerateTransactionID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for range 100 {
		id := generateTransactionID()

		if id == "" {
			t.Error("Transaction ID should not be empty")
		}

		if !strings.HasPrefix(id, "tx_") {
			t.Errorf("Transaction ID should start with 'tx_', got %s", id)
		}

		// Check for uniqueness
		if ids[id] {
			t.Errorf("Duplicate transaction ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	testDir := "./test_file_exists"
	os.MkdirAll(testDir, 0o755)
	defer os.RemoveAll(testDir)

	existingFile := filepath.Join(testDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test existing file
	if !fileExists(existingFile) {
		t.Error("fileExists should return true for existing file")
	}

	// Test non-existing file
	nonExistingFile := filepath.Join(testDir, "nonexisting.txt")
	if fileExists(nonExistingFile) {
		t.Error("fileExists should return false for non-existing file")
	}
}

func TestTransactionLog_ConcurrentAccess(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	// Test concurrent access to transaction log
	numGoroutines := 10
	done := make(chan bool, numGoroutines)

	for i := range numGoroutines {
		go func(index int) {
			defer func() { done <- true }()

			// Begin transaction
			txID := tl.BeginTransaction(
				fmt.Sprintf("/test/file%d.txt", index),
				fmt.Sprintf("/test/file%d.txt.tmp", index),
				fmt.Sprintf("/test/file%d.txt.backup", index),
			)

			// Complete or fail randomly
			if index%2 == 0 {
				tl.CompleteTransaction(txID)
			} else {
				tl.FailTransaction(txID, fmt.Errorf("test error %d", index))
			}

			// Get summary (concurrent read)
			summary := tl.GetSummary()
			if summary.TotalTransactions <= 0 {
				t.Errorf("Summary should show transactions")
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for range numGoroutines {
		<-done
	}

	// Final check
	summary := tl.GetSummary()
	if summary.TotalTransactions != numGoroutines {
		t.Errorf("Expected %d total transactions, got %d", numGoroutines, summary.TotalTransactions)
	}
}

func TestTransactionLog_LogFile(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	tl := NewTransactionLog()

	// Create a transaction to trigger logging
	txID := tl.BeginTransaction("/test/target.txt", "/test/target.txt.tmp", "/test/target.txt.backup")
	tl.CompleteTransaction(txID)

	// Check that log file exists and has content
	if _, err := os.Stat(tl.logFile); os.IsNotExist(err) {
		t.Errorf("Log file should exist: %s", tl.logFile)
	}

	// Read log file content
	content, err := os.ReadFile(tl.logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// Should contain transaction data
	if len(content) == 0 {
		t.Error("Log file should not be empty")
	}

	// Should contain transaction ID
	if !strings.Contains(string(content), txID) {
		t.Error("Log file should contain transaction ID")
	}
}

func TestCleanupTransaction(t *testing.T) {
	os.RemoveAll("./.morfx")
	defer os.RemoveAll("./.morfx")

	// Create test files
	testDir := "./test_cleanup"
	os.MkdirAll(testDir, 0o755)
	defer os.RemoveAll(testDir)

	backupPath := filepath.Join(testDir, "backup.txt")
	if err := os.WriteFile(backupPath, []byte("backup content"), 0o644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	tl := NewTransactionLog()
	txID := tl.BeginTransaction("/test/target.txt", "", backupPath)
	tl.CompleteTransaction(txID)

	// Verify transaction exists
	tl.mutex.RLock()
	_, exists := tl.transactions[txID]
	tl.mutex.RUnlock()

	if !exists {
		t.Error("Transaction should exist before cleanup")
	}

	// Manually trigger cleanup (normally done with delay)
	tl.cleanupTransaction(txID)

	// Check transaction was removed
	tl.mutex.RLock()
	_, exists = tl.transactions[txID]
	tl.mutex.RUnlock()

	if exists {
		t.Error("Transaction should be removed after cleanup")
	}

	// Check backup file was removed for completed transaction
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup file should be removed for completed transaction")
	}
}
