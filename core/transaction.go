package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// TransactionOperation represents a single operation in a transaction
type TransactionOperation struct {
	Type       string    `json:"type"` // "modify", "create", "delete"
	FilePath   string    `json:"file_path"`
	BackupPath string    `json:"backup_path"` // For rollback
	Checksum   string    `json:"checksum"`    // Original file checksum
	Timestamp  time.Time `json:"timestamp"`
	Completed  bool      `json:"completed"`
	Error      string    `json:"error,omitempty"`
}

// TransactionLog represents a complete transaction
type TransactionLog struct {
	ID          string                 `json:"id"`
	Started     time.Time              `json:"started"`
	Completed   time.Time              `json:"completed"`
	Operations  []TransactionOperation `json:"operations"`
	Status      string                 `json:"status"` // "pending", "committed", "rolled_back"
	Description string                 `json:"description"`
}

// TransactionManager handles transaction logging and rollback
type TransactionManager struct {
	logDir       string
	currentTx    *TransactionLog
	atomicWriter *AtomicWriter
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(logDir string, atomicWriter *AtomicWriter) *TransactionManager {
	os.MkdirAll(logDir, 0o755)

	return &TransactionManager{
		logDir:       logDir,
		atomicWriter: atomicWriter,
	}
}

// BeginTransaction starts a new transaction
func (tm *TransactionManager) BeginTransaction(description string) (*TransactionLog, error) {
	if tm.currentTx != nil {
		return nil, fmt.Errorf("transaction already in progress: %s", tm.currentTx.ID)
	}

	txID := fmt.Sprintf("tx_%d_%d", time.Now().Unix(), os.Getpid())

	tx := &TransactionLog{
		ID:          txID,
		Started:     time.Now(),
		Operations:  make([]TransactionOperation, 0),
		Status:      "pending",
		Description: description,
	}

	tm.currentTx = tx

	// Write initial transaction log
	if err := tm.writeTransactionLog(tx); err != nil {
		tm.currentTx = nil
		return nil, fmt.Errorf("failed to write transaction log: %w", err)
	}

	return tx, nil
}

// AddOperation records a file operation in the current transaction
func (tm *TransactionManager) AddOperation(opType, filePath string) (*TransactionOperation, error) {
	if tm.currentTx == nil {
		return nil, fmt.Errorf("no active transaction")
	}

	op := TransactionOperation{
		Type:      opType,
		FilePath:  filePath,
		Timestamp: time.Now(),
		Completed: false,
	}

	// Generate checksum for existing files
	if opType == "modify" || opType == "delete" {
		if _, err := os.Stat(filePath); err == nil {
			checksum, err := generateFileChecksum(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to generate checksum: %w", err)
			}
			op.Checksum = checksum

			// Create backup for modify operations
			if opType == "modify" {
				backupPath := tm.generateBackupPath(filePath)
				if err := tm.createBackup(filePath, backupPath); err != nil {
					return nil, fmt.Errorf("failed to create backup: %w", err)
				}
				op.BackupPath = backupPath
			}
		}
	}

	tm.currentTx.Operations = append(tm.currentTx.Operations, op)

	// Update transaction log
	if err := tm.writeTransactionLog(tm.currentTx); err != nil {
		return nil, fmt.Errorf("failed to update transaction log: %w", err)
	}

	return &op, nil
}

// CompleteOperation marks an operation as completed
func (tm *TransactionManager) CompleteOperation(filePath string, err error) error {
	if tm.currentTx == nil {
		return fmt.Errorf("no active transaction")
	}

	// Find the operation
	for i := range tm.currentTx.Operations {
		op := &tm.currentTx.Operations[i]
		if op.FilePath == filePath && !op.Completed {
			op.Completed = true
			if err != nil {
				op.Error = err.Error()
			}

			// Update transaction log
			return tm.writeTransactionLog(tm.currentTx)
		}
	}

	return fmt.Errorf("operation not found for file: %s", filePath)
}

// CommitTransaction marks transaction as successfully completed
func (tm *TransactionManager) CommitTransaction() error {
	if tm.currentTx == nil {
		return fmt.Errorf("no active transaction")
	}

	// Check if all operations completed successfully
	for _, op := range tm.currentTx.Operations {
		if !op.Completed || op.Error != "" {
			return fmt.Errorf("cannot commit transaction with failed operations")
		}
	}

	tm.currentTx.Status = "committed"
	tm.currentTx.Completed = time.Now()

	err := tm.writeTransactionLog(tm.currentTx)
	tm.currentTx = nil // Clear current transaction

	return err
}

// RollbackTransaction reverts all operations in current transaction
func (tm *TransactionManager) RollbackTransaction() error {
	if tm.currentTx == nil {
		return fmt.Errorf("no active transaction")
	}

	var rollbackErrors []string

	// Process operations in reverse order
	for i := len(tm.currentTx.Operations) - 1; i >= 0; i-- {
		op := tm.currentTx.Operations[i]

		if !op.Completed {
			continue // Skip incomplete operations
		}

		if err := tm.rollbackOperation(op); err != nil {
			rollbackErrors = append(rollbackErrors,
				fmt.Sprintf("failed to rollback %s: %v", op.FilePath, err))
		}
	}

	tm.currentTx.Status = "rolled_back"
	tm.currentTx.Completed = time.Now()

	if err := tm.writeTransactionLog(tm.currentTx); err != nil {
		rollbackErrors = append(rollbackErrors,
			fmt.Sprintf("failed to update transaction log: %v", err))
	}

	tm.currentTx = nil // Clear current transaction

	if len(rollbackErrors) > 0 {
		return fmt.Errorf("rollback completed with errors: %v", rollbackErrors)
	}

	return nil
}

// rollbackOperation reverts a single operation
func (tm *TransactionManager) rollbackOperation(op TransactionOperation) error {
	switch op.Type {
	case "modify":
		// Restore from backup
		if op.BackupPath == "" {
			return fmt.Errorf("no backup path for modify operation")
		}

		if _, err := os.Stat(op.BackupPath); err != nil {
			return fmt.Errorf("backup file not found: %s", op.BackupPath)
		}

		content, err := os.ReadFile(op.BackupPath)
		if err != nil {
			return fmt.Errorf("failed to read backup: %w", err)
		}

		return tm.atomicWriter.WriteFile(op.FilePath, string(content))

	case "create":
		// Delete the created file
		if _, err := os.Stat(op.FilePath); err == nil {
			return os.Remove(op.FilePath)
		}
		return nil // File doesn't exist, nothing to do

	case "delete":
		// Can't easily rollback delete without backup
		return fmt.Errorf("cannot rollback delete operation for %s", op.FilePath)

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}
}

// LoadTransaction loads a transaction from log file
func (tm *TransactionManager) LoadTransaction(txID string) (*TransactionLog, error) {
	logPath := filepath.Join(tm.logDir, txID+".json")

	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction log: %w", err)
	}

	var tx TransactionLog
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to parse transaction log: %w", err)
	}

	return &tx, nil
}

// ListPendingTransactions returns all pending transactions
func (tm *TransactionManager) ListPendingTransactions() ([]TransactionLog, error) {
	files, err := os.ReadDir(tm.logDir)
	if err != nil {
		return nil, err
	}

	var pending []TransactionLog
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			txID := file.Name()[:len(file.Name())-5] // Remove .json
			tx, err := tm.LoadTransaction(txID)
			if err != nil {
				continue // Skip corrupted logs
			}

			if tx.Status == "pending" {
				pending = append(pending, *tx)
			}
		}
	}

	return pending, nil
}

// CleanupOldTransactions removes old completed transaction logs
func (tm *TransactionManager) CleanupOldTransactions(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	files, err := os.ReadDir(tm.logDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			txID := file.Name()[:len(file.Name())-5]
			tx, err := tm.LoadTransaction(txID)
			if err != nil {
				continue
			}

			// Remove completed transactions older than cutoff
			if tx.Status != "pending" && tx.Completed.Before(cutoff) {
				logPath := filepath.Join(tm.logDir, file.Name())
				os.Remove(logPath)

				// Also remove backup files for this transaction
				for _, op := range tx.Operations {
					if op.BackupPath != "" {
						os.Remove(op.BackupPath)
					}
				}
			}
		}
	}

	return nil
}

// writeTransactionLog writes transaction to disk
func (tm *TransactionManager) writeTransactionLog(tx *TransactionLog) error {
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}

	logPath := filepath.Join(tm.logDir, tx.ID+".json")
	return os.WriteFile(logPath, data, 0o644)
}

// generateBackupPath creates a unique backup path
func (tm *TransactionManager) generateBackupPath(filePath string) string {
	timestamp := time.Now().Format("20060102-150405")
	txID := "unknown"
	if tm.currentTx != nil {
		txID = tm.currentTx.ID
	}

	dir := filepath.Dir(filePath)
	name := filepath.Base(filePath)

	return filepath.Join(dir, fmt.Sprintf(".morfx-backup-%s-%s-%s",
		name, txID, timestamp))
}

// createBackup creates a backup file
func (tm *TransactionManager) createBackup(originalPath, backupPath string) error {
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, content, 0o644)
}

// generateFileChecksum creates SHA256 hash of file content
func generateFileChecksum(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash), nil
}
