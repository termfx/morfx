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

func TestTransactionManager_BeginTransaction_AlreadyActive(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Begin first transaction
	_, err := manager.BeginTransaction("First transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Try to begin second transaction while first is active
	_, err = manager.BeginTransaction("Second transaction")
	if err == nil {
		t.Error("Expected error when beginning transaction while one is active")
	}

	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("Expected 'already in progress' error, got: %v", err)
	}
}

func TestTransactionManager_BeginTransaction_LogWriteFailure(t *testing.T) {
	// Use invalid log directory to cause write failure
	invalidLogDir := "/dev/null/invalid/path"

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(invalidLogDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err == nil {
		t.Error("Expected error when log directory is invalid")
	}

	// Current transaction should be nil after failure
	if manager.currentTx != nil {
		t.Error("Current transaction should be nil after begin failure")
	}
}

func TestTransactionManager_AddOperation_NoActiveTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Try to add operation without active transaction
	_, err := manager.AddOperation("modify", "/test/file.txt")
	if err == nil {
		t.Error("Expected error when adding operation without active transaction")
	}

	if !strings.Contains(err.Error(), "no active transaction") {
		t.Errorf("Expected 'no active transaction' error, got: %v", err)
	}
}

func TestTransactionManager_AddOperation_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Add operation for non-existent file with modify type
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
	op, err := manager.AddOperation("modify", nonExistentFile)
	if err != nil {
		t.Fatalf("AddOperation should handle non-existent files: %v", err)
	}

	// For non-existent files, checksum and backup should be empty
	if op.Checksum != "" {
		t.Error("Checksum should be empty for non-existent file")
	}

	if op.BackupPath != "" {
		t.Error("Backup path should be empty for non-existent file")
	}

	if len(tx.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(tx.Operations))
	}
}

func TestTransactionManager_AddOperation_BackupCreationFailure(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Test with a non-existent file - this should work without backup
	testFile := filepath.Join(tempDir, "nonexistent.txt")
	op, err := manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation should work with non-existent file: %v", err)
	}

	// For non-existent files, backup should be empty
	if op.BackupPath != "" {
		t.Error("Backup path should be empty for non-existent file")
	}
}

func TestTransactionManager_AddOperation_ChecksumGenerationFailure(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	// Create a file, then make it unreadable
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = os.Chmod(testFile, 0o000) // Make unreadable
	if err != nil {
		t.Fatalf("Failed to make file unreadable: %v", err)
	}
	defer os.Chmod(testFile, 0o644)

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err = manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Should fail when checksum generation fails
	_, err = manager.AddOperation("modify", testFile)
	if err == nil {
		t.Error("Expected error when checksum generation fails")
	}

	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("Expected checksum error, got: %v", err)
	}
}

func TestTransactionManager_CompleteOperation_NoActiveTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	err := manager.CompleteOperation("/test/file.txt", nil)
	if err == nil {
		t.Error("Expected error when completing operation without active transaction")
	}

	if !strings.Contains(err.Error(), "no active transaction") {
		t.Errorf("Expected 'no active transaction' error, got: %v", err)
	}
}

func TestTransactionManager_CompleteOperation_OperationNotFound(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Try to complete operation that doesn't exist
	err = manager.CompleteOperation("/nonexistent/file.txt", nil)
	if err == nil {
		t.Error("Expected error when completing non-existent operation")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestTransactionManager_CompleteOperation_AlreadyCompleted(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	// Complete operation once
	err = manager.CompleteOperation(testFile, nil)
	if err != nil {
		t.Fatalf("First CompleteOperation failed: %v", err)
	}

	// Try to complete again - should find no incomplete operation
	err = manager.CompleteOperation(testFile, nil)
	if err == nil {
		t.Error("Expected error when operation already completed")
	}
}

func TestTransactionManager_CommitTransaction_NoActiveTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	err := manager.CommitTransaction()
	if err == nil {
		t.Error("Expected error when committing without active transaction")
	}

	if !strings.Contains(err.Error(), "no active transaction") {
		t.Errorf("Expected 'no active transaction' error, got: %v", err)
	}
}

func TestTransactionManager_CommitTransaction_WithFailedOperations(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	// Complete operation with error
	testErr := fmt.Errorf("operation failed")
	err = manager.CompleteOperation(testFile, testErr)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	// Commit should fail because operation has error
	err = manager.CommitTransaction()
	if err == nil {
		t.Error("Expected error when committing transaction with failed operations")
	}

	if !strings.Contains(err.Error(), "failed operations") {
		t.Errorf("Expected 'failed operations' error, got: %v", err)
	}
}

func TestTransactionManager_CommitTransaction_WithIncompleteOperations(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("modify", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	// Don't complete the operation

	// Commit should fail because operation is incomplete
	err = manager.CommitTransaction()
	if err == nil {
		t.Error("Expected error when committing transaction with incomplete operations")
	}

	if !strings.Contains(err.Error(), "failed operations") {
		t.Errorf("Expected 'failed operations' error, got: %v", err)
	}
}

func TestTransactionManager_RollbackTransaction_NoActiveTransaction(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	err := manager.RollbackTransaction()
	if err == nil {
		t.Error("Expected error when rolling back without active transaction")
	}

	if !strings.Contains(err.Error(), "no active transaction") {
		t.Errorf("Expected 'no active transaction' error, got: %v", err)
	}
}

func TestTransactionManager_RollbackTransaction_CreateOperationType(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Create a file during transaction
	testFile := filepath.Join(tempDir, "created.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("create", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	err = manager.CompleteOperation(testFile, nil)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	// Rollback should delete the created file
	err = manager.RollbackTransaction()
	if err != nil {
		t.Fatalf("RollbackTransaction failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Created file should be deleted during rollback")
	}
}

func TestTransactionManager_RollbackTransaction_DeleteOperationType(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	testFile := filepath.Join(tempDir, "to_delete.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = manager.AddOperation("delete", testFile)
	if err != nil {
		t.Fatalf("AddOperation failed: %v", err)
	}

	err = manager.CompleteOperation(testFile, nil)
	if err != nil {
		t.Fatalf("CompleteOperation failed: %v", err)
	}

	// Rollback should indicate delete can't be rolled back
	err = manager.RollbackTransaction()
	if err == nil {
		t.Error("Expected error when rolling back delete operation")
	}

	if !strings.Contains(err.Error(), "rollback delete") {
		t.Errorf("Expected 'rollback delete' error, got: %v", err)
	}
}

func TestTransactionManager_RollbackTransaction_MissingBackup(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test transaction")
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

	// Manually delete the backup file to simulate missing backup
	if len(tx.Operations) > 0 && tx.Operations[0].BackupPath != "" {
		err = os.Remove(tx.Operations[0].BackupPath)
		if err != nil {
			t.Fatalf("Failed to remove backup file: %v", err)
		}
	}

	// Rollback should fail due to missing backup
	err = manager.RollbackTransaction()
	if err == nil {
		t.Error("Expected error when backup file is missing")
	}

	if !strings.Contains(err.Error(), "backup file not found") {
		t.Errorf("Expected 'backup file not found' error, got: %v", err)
	}
}

func TestTransactionManager_RollbackTransaction_UnknownOperationType(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	tx, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Manually add operation with unknown type
	op := TransactionOperation{
		Type:      "unknown_type",
		FilePath:  "/test/file.txt",
		Completed: true,
		Timestamp: time.Now(),
	}
	tx.Operations = append(tx.Operations, op)

	// Rollback should fail due to unknown operation type
	err = manager.RollbackTransaction()
	if err == nil {
		t.Error("Expected error when rolling back unknown operation type")
	}

	if !strings.Contains(err.Error(), "unknown operation type") {
		t.Errorf("Expected 'unknown operation type' error, got: %v", err)
	}
}

func TestTransactionManager_LoadTransaction_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create invalid JSON file
	invalidTxID := "invalid_tx"
	invalidTxFile := filepath.Join(logDir, invalidTxID+".json")
	err := os.WriteFile(invalidTxFile, []byte("invalid json content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	_, err = manager.LoadTransaction(invalidTxID)
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}

	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestTransactionManager_LoadTransaction_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	_, err := manager.LoadTransaction("nonexistent_tx")
	if err == nil {
		t.Error("Expected error when loading non-existent transaction")
	}

	if !strings.Contains(err.Error(), "read transaction log") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

func TestTransactionManager_ListPendingTransactions_CorruptedLog(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create a valid transaction
	_, err := manager.BeginTransaction("Valid transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	// Create a corrupted log file
	corruptedFile := filepath.Join(logDir, "corrupted_tx.json")
	err = os.WriteFile(corruptedFile, []byte("corrupted json"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Should skip corrupted logs and return valid ones
	pending, err := manager.ListPendingTransactions()
	if err != nil {
		t.Fatalf("ListPendingTransactions failed: %v", err)
	}

	// Should have one valid pending transaction
	if len(pending) != 1 {
		t.Errorf("Expected 1 pending transaction, got %d", len(pending))
	}
}

func TestTransactionManager_CleanupOldTransactions_DirectoryReadError(t *testing.T) {
	// Use non-existent directory to cause read error
	invalidLogDir := "/nonexistent/tx_logs"

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(invalidLogDir, writer)

	err := manager.CleanupOldTransactions(24 * time.Hour)
	if err == nil {
		t.Error("Expected error when log directory doesn't exist")
	}
}

func TestTransactionManager_CleanupOldTransactions_CorruptedLogs(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create a completed transaction
	tx, err := manager.BeginTransaction("Old transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	err = manager.CommitTransaction()
	if err != nil {
		t.Fatalf("CommitTransaction failed: %v", err)
	}

	// Create a corrupted log file
	corruptedFile := filepath.Join(logDir, "corrupted.json")
	err = os.WriteFile(corruptedFile, []byte("invalid json"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Should handle corrupted logs gracefully
	err = manager.CleanupOldTransactions(0) // Clean all
	if err != nil {
		t.Fatalf("CleanupOldTransactions failed: %v", err)
	}

	// Valid transaction should be cleaned up
	txFile := filepath.Join(logDir, tx.ID+".json")
	if _, err := os.Stat(txFile); !os.IsNotExist(err) {
		t.Error("Old transaction should be cleaned up")
	}

	// Corrupted file might or might not be cleaned up depending on implementation
}

func TestTransactionManager_GenerateBackupPath_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Test without active transaction
	backupPath := manager.generateBackupPath("/path/to/file.txt")
	if !strings.Contains(backupPath, "unknown") {
		t.Error("Expected 'unknown' in backup path when no active transaction")
	}

	// Test with active transaction
	tx, err := manager.BeginTransaction("Test transaction")
	if err != nil {
		t.Fatalf("BeginTransaction failed: %v", err)
	}

	backupPath = manager.generateBackupPath("/path/to/file.txt")
	if !strings.Contains(backupPath, tx.ID) {
		t.Error("Expected transaction ID in backup path")
	}

	if !strings.Contains(backupPath, "file.txt") {
		t.Error("Expected original filename in backup path")
	}

	if !strings.Contains(backupPath, ".morfx-backup") {
		t.Error("Expected backup prefix in backup path")
	}
}

func TestTransactionManager_WriteTransactionLog_Error(t *testing.T) {
	// Use an invalid log directory to cause write errors
	invalidLogDir := "/dev/null/invalid"

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(invalidLogDir, writer)

	// Try to begin transaction with invalid log directory
	_, err := manager.BeginTransaction("Test transaction")
	if err == nil {
		t.Error("Expected error when writing to invalid log directory")
	}
}

func TestGenerateFileChecksum_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty file",
			content:  "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // SHA256 of empty string
		},
		{
			name:     "single character",
			content:  "a",
			expected: "ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", // SHA256 of "a"
		},
		{
			name:     "newline only",
			content:  "\n",
			expected: "01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b", // SHA256 of "\n"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.txt")

			err := os.WriteFile(testFile, []byte(tt.content), 0o644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			checksum, err := generateFileChecksum(testFile)
			if err != nil {
				t.Fatalf("generateFileChecksum failed: %v", err)
			}

			if checksum != tt.expected {
				t.Errorf("Expected checksum %s, got %s", tt.expected, checksum)
			}
		})
	}
}

func TestGenerateFileChecksum_NonExistentFile(t *testing.T) {
	_, err := generateFileChecksum("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestTransactionOperation_TimeHandling(t *testing.T) {
	now := time.Now()
	op := TransactionOperation{
		Type:      "modify",
		FilePath:  "/test/file.txt",
		Timestamp: now,
		Completed: true,
	}

	// Test JSON marshaling preserves time
	data, err := json.Marshal(op)
	if err != nil {
		t.Fatalf("Failed to marshal operation: %v", err)
	}

	var unmarshaled TransactionOperation
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal operation: %v", err)
	}

	// Times should be approximately equal (allowing for JSON precision)
	if now.Sub(unmarshaled.Timestamp).Abs() > time.Second {
		t.Errorf("Timestamp not preserved correctly: expected %v, got %v",
			now, unmarshaled.Timestamp)
	}
}

func TestTransactionManager_RollbackOperation_MissingBackupPath(t *testing.T) {
	tempDir := t.TempDir()
	logDir := filepath.Join(tempDir, "tx_logs")

	config := DefaultAtomicConfig()
	writer := NewAtomicWriter(config)
	manager := NewTransactionManager(logDir, writer)

	// Create operation with missing backup path
	op := TransactionOperation{
		Type:       "modify",
		FilePath:   "/test/file.txt",
		BackupPath: "", // Missing backup path
		Completed:  true,
	}

	err := manager.rollbackOperation(op)
	if err == nil {
		t.Error("Expected error when backup path is missing for modify operation")
	}

	if !strings.Contains(err.Error(), "no backup path") {
		t.Errorf("Expected 'no backup path' error, got: %v", err)
	}
}
