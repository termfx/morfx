package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewTransactionManager(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)

	manager := NewTransactionManager(logDir, writer)

	if manager == nil {
		t.Fatal("NewTransactionManager returned nil")
	}

	if manager.logDir != logDir {
		t.Errorf("Expected logDir %s, got %s", logDir, manager.logDir)
	}

	if manager.atomicWriter != writer {
		t.Error("AtomicWriter not set correctly")
	}

	// Verify log directory was created
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Error("Log directory was not created")
	}
}

func TestTransactionManager_BeginTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	description := "Test transaction"
	tx, err := manager.BeginTransaction(description)
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	if tx == nil {
		t.Fatal("BeginTransaction returned nil transaction")
	}

	if tx.ID == "" {
		t.Error("Transaction ID is empty")
	}

	if tx.Description != description {
		t.Errorf("Expected description '%s', got '%s'", description, tx.Description)
	}

	if tx.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", tx.Status)
	}

	if tx.Started.IsZero() {
		t.Error("Transaction start time not set")
	}

	if len(tx.Operations) != 0 {
		t.Error("New transaction should have no operations")
	}

	// Verify transaction was saved to disk
	txFile := filepath.Join(logDir, tx.ID+".json")
	if _, err := os.Stat(txFile); os.IsNotExist(err) {
		t.Error("Transaction log file was not created")
	}
}

func TestTransactionManager_AddOperation(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test adding operations")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("original content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add modify operation
	op, err := manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	if len(tx.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(tx.Operations))
	}
	if op.Type != "modify" {
		t.Errorf("Expected operation type 'modify', got '%s'", op.Type)
	}

	if op.FilePath != testFile {
		t.Errorf("Expected file path '%s', got '%s'", testFile, op.FilePath)
	}

	if op.BackupPath == "" {
		t.Error("Backup path not set")
	}

	if op.Checksum == "" {
		t.Error("Checksum not set")
	}

	if op.Completed {
		t.Error("Operation should not be completed yet")
	}

	// Verify backup file was created
	if _, err := os.Stat(op.BackupPath); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}
}

func TestTransactionManager_CompleteOperation(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test completing operations")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("original"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add and complete operation
	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	err = manager.CompleteOperation(testFile, nil)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	op := tx.Operations[0]
	if !op.Completed {
		t.Error("Operation should be marked as completed")
	}

	if op.Error != "" {
		t.Errorf("Expected no error, got '%s'", op.Error)
	}
}

func TestTransactionManager_CompleteOperation_WithError(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test operation error")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("original"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	errorMsg := "test error occurred"
	testErr := fmt.Errorf("%s", errorMsg)
	err = manager.CompleteOperation(testFile, testErr)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	op := tx.Operations[0]
	if !op.Completed {
		t.Error("Operation should be marked as completed even with error")
	}

	if op.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, op.Error)
	}
}

func TestTransactionManager_CommitTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test commit")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("original"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	err = manager.CompleteOperation(testFile, nil)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	err = manager.CommitTransaction()
	if err != nil {
		t.Fatalf("CommitTransaction failed: %v", err)
	}

	if tx.Status != "committed" {
		t.Errorf("Expected status 'committed', got '%s'", tx.Status)
	}

	if tx.Completed.IsZero() {
		t.Error("Transaction completion time not set")
	}

	// Verify transaction was committed (backup cleanup may or may not happen)
	// The implementation doesn't automatically clean up backups on commit
}

func TestTransactionManager_RollbackTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test rollback")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "original content"
	err = os.WriteFile(testFile, []byte(originalContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	// Modify the file to simulate a change
	modifiedContent := "modified content"
	err = os.WriteFile(testFile, []byte(modifiedContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	err = manager.CompleteOperation(testFile, nil)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	// Rollback should restore original content
	err = manager.RollbackTransaction()
	if err != nil {
		t.Fatalf("RollbackTransaction failed: %v", err)
	}

	if tx.Status != "rolled_back" {
		t.Errorf("Expected status 'rolled_back', got '%s'", tx.Status)
	}

	// Verify file content was restored
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file after rollback: %v", err)
	}

	if string(content) != originalContent {
		t.Errorf("Expected content '%s', got '%s'", originalContent, string(content))
	}
}

func TestTransactionManager_LoadTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create a transaction
	tx, err := manager.BeginTransaction("Test load")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	originalID := tx.ID

	// Load the transaction
	loadedTx, err := manager.LoadTransaction(originalID)
	if err != nil {
		t.Fatalf("LoadTransaction failed: %v", err)
	}

	if loadedTx.ID != originalID {
		t.Errorf("Expected ID '%s', got '%s'", originalID, loadedTx.ID)
	}

	if loadedTx.Description != tx.Description {
		t.Errorf("Expected description '%s', got '%s'", tx.Description, loadedTx.Description)
	}
}

func TestTransactionManager_LoadTransaction_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.LoadTransaction("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent transaction")
	}
}

func TestTransactionManager_ListPendingTransactions(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create and commit first transaction
	_, err := manager.BeginTransaction("Committed transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	err = manager.CommitTransaction()
	if err != nil {
		t.Fatalf("CommitTransaction failed: %v", err)
	}

	// Create second transaction and leave it pending
	tx2, err := manager.BeginTransaction("Pending transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// List pending transactions
	pending, err := manager.ListPendingTransactions()
	if err != nil {
		t.Fatalf("ListPendingTransactions failed: %v", err)
	}

	// Should have tx2 as pending
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending transaction, got %d", len(pending))
	}

	if len(pending) > 0 && pending[0].ID != tx2.ID {
		t.Errorf("Expected pending transaction %s, got %s", tx2.ID, pending[0].ID)
	}
}

func TestTransactionManager_CleanupOldTransactions(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create and commit a transaction
	tx, err := manager.BeginTransaction("Old transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	err = manager.CommitTransaction()
	if err != nil {
		t.Fatalf("CommitTransaction failed: %v", err)
	}

	// Verify transaction file exists
	txFile := filepath.Join(logDir, tx.ID+".json")
	if _, err := os.Stat(txFile); os.IsNotExist(err) {
		t.Error("Transaction file should exist before cleanup")
	}

	// Cleanup with 0 duration (should clean all)
	err = manager.CleanupOldTransactions(0)
	if err != nil {
		t.Fatalf("CleanupOldTransactions failed: %v", err)
	}

	// Verify transaction file was removed
	if _, err := os.Stat(txFile); !os.IsNotExist(err) {
		t.Error("Transaction file should be removed after cleanup")
	}
}

func TestTransactionManager_DeleteOperation(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test delete operation")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "to_delete.txt")
	err = os.WriteFile(testFile, []byte("file to delete"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Add delete operation
	_, err = manager.AddOperation("delete", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	op := tx.Operations[0]
	if op.Type != "delete" {
		t.Errorf("Expected operation type 'delete', got '%s'", op.Type)
	}

	// For delete operations, the implementation doesn't create backups during AddOperation
	// Checksum should be recorded for rollback verification
	if op.Checksum == "" {
		t.Error("Checksum should be set for delete operations")
	}
}

func TestTransactionManager_CreateOperation(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test create operation")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Add create operation for non-existent file
	newFile := filepath.Join(tempDir, "new_file.txt")
	_, err = manager.AddOperation("create", newFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	op := tx.Operations[0]
	if op.Type != "create" {
		t.Errorf("Expected operation type 'create', got '%s'", op.Type)
	}

	// For create operations, no backup should be needed
	if op.BackupPath != "" {
		t.Error("Backup path should be empty for create operations")
	}

	if op.Checksum != "" {
		t.Error("Checksum should be empty for create operations (no original file)")
	}
}

func TestTransactionOperation_JSONSerialization(t *testing.T) {
	op := TransactionOperation{
		Type:       "modify",
		FilePath:   "/path/to/file.txt",
		BackupPath: "/path/to/backup.txt",
		Checksum:   "abc123",
		Timestamp:  time.Now(),
		Completed:  true,
		Error:      "test error",
	}

	// Marshal to JSON
	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("Failed to marshal operation: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled TransactionOperation
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal operation: %v", err)
	}

	// Verify fields
	if unmarshaled.Type != op.Type {
		t.Errorf("Type mismatch: expected %s, got %s", op.Type, unmarshaled.Type)
	}

	if unmarshaled.FilePath != op.FilePath {
		t.Errorf("FilePath mismatch: expected %s, got %s", op.FilePath, unmarshaled.FilePath)
	}

	if unmarshaled.Completed != op.Completed {
		t.Errorf("Completed mismatch: expected %t, got %t", op.Completed, unmarshaled.Completed)
	}
}

func TestTransactionLog_JSONSerialization(t *testing.T) {
	now := time.Now()
	log := TransactionLog{
		ID:          "test-tx-123",
		Started:     now,
		Completed:   now.Add(time.Minute),
		Status:      "committed",
		Description: "Test transaction",
		Operations: []TransactionOperation{
			{
				Type:      "modify",
				FilePath:  "/test/file.txt",
				Completed: true,
				Timestamp: now,
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("Failed to marshal transaction log: %v", err)
	}

	// Unmarshal from JSON
	var unmarshaled TransactionLog
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal transaction log: %v", err)
	}

	// Verify fields
	if unmarshaled.ID != log.ID {
		t.Errorf("ID mismatch: expected %s, got %s", log.ID, unmarshaled.ID)
	}

	if unmarshaled.Description != log.Description {
		t.Errorf("Description mismatch: expected %s, got %s", log.Description, unmarshaled.Description)
	}

	if len(unmarshaled.Operations) != len(log.Operations) {
		t.Errorf("Operations length mismatch: expected %d, got %d", len(log.Operations), len(unmarshaled.Operations))
	}
}

func TestTransactionManager_InvalidOperationIndex(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test invalid operation")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Try to complete operation for non-existent file
	err = manager.CompleteOperation("/nonexistent/file.txt", nil)
	if err == nil {
		t.Error("Expected error for non-existent file operation")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error about operation not found, got: %v", err)
	}
}
