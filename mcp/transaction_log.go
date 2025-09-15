package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TransactionLog manages transaction logging for rollback capability
type TransactionLog struct {
	transactions map[string]*Transaction
	mutex        sync.RWMutex
	logFile      string
}

// NewTransactionLog creates a new transaction log
func NewTransactionLog() *TransactionLog {
	logDir := "./.morfx/transactions"
	os.MkdirAll(logDir, 0o755)

	logFile := filepath.Join(logDir, fmt.Sprintf("tx_%d.log", time.Now().Unix()))

	return &TransactionLog{
		transactions: make(map[string]*Transaction),
		logFile:      logFile,
	}
}

// BeginTransaction starts a new transaction
func (tl *TransactionLog) BeginTransaction(targetPath, tmpPath, backupPath string) string {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()

	txID := generateTransactionID()
	tx := &Transaction{
		ID:         txID,
		TargetPath: targetPath,
		TmpPath:    tmpPath,
		BackupPath: backupPath,
		Status:     "pending",
		StartTime:  time.Now(),
	}

	tl.transactions[txID] = tx
	tl.logTransaction(tx)

	return txID
}

// CompleteTransaction marks a transaction as completed
func (tl *TransactionLog) CompleteTransaction(txID string) {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()

	tx, exists := tl.transactions[txID]
	if !exists {
		return
	}

	tx.Status = "completed"
	tx.EndTime = time.Now()
	tl.logTransaction(tx)

	// Clean up completed transaction after a delay
	go func() {
		time.Sleep(5 * time.Minute)
		tl.cleanupTransaction(txID)
	}()
}

// FailTransaction marks a transaction as failed and triggers rollback
func (tl *TransactionLog) FailTransaction(txID string, reason error) error {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()

	tx, exists := tl.transactions[txID]
	if !exists {
		return fmt.Errorf("transaction not found: %s", txID)
	}

	tx.Status = "failed"
	tx.Error = reason.Error()
	tx.EndTime = time.Now()
	tl.logTransaction(tx)

	return tl.rollbackTransaction(tx)
}

// RollbackTransaction performs rollback for a specific transaction
func (tl *TransactionLog) RollbackTransaction(txID string) error {
	tl.mutex.RLock()
	tx, exists := tl.transactions[txID]
	tl.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("transaction not found: %s", txID)
	}

	return tl.rollbackTransaction(tx)
}

// rollbackTransaction performs the actual rollback logic
func (tl *TransactionLog) rollbackTransaction(tx *Transaction) error {
	// If we have a backup, restore it
	if tx.BackupPath != "" && fileExists(tx.BackupPath) {
		if err := os.Rename(tx.BackupPath, tx.TargetPath); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}
	} else {
		// No backup available, just remove the target if it exists
		if fileExists(tx.TargetPath) {
			if err := os.Remove(tx.TargetPath); err != nil {
				return fmt.Errorf("failed to remove target file: %w", err)
			}
		}
	}

	// Clean up temporary file if it still exists
	if tx.TmpPath != "" && fileExists(tx.TmpPath) {
		os.Remove(tx.TmpPath)
	}

	tx.Status = "rolled_back"
	tx.EndTime = time.Now()
	tl.logTransaction(tx)

	return nil
}

// GetPendingTransactions returns all pending transactions
func (tl *TransactionLog) GetPendingTransactions() []*Transaction {
	tl.mutex.RLock()
	defer tl.mutex.RUnlock()

	var pending []*Transaction
	for _, tx := range tl.transactions {
		if tx.Status == "pending" {
			pending = append(pending, tx)
		}
	}

	return pending
}

// RollbackAll rolls back all pending transactions
func (tl *TransactionLog) RollbackAll() error {
	pending := tl.GetPendingTransactions()

	var errors []string
	for _, tx := range pending {
		if err := tl.rollbackTransaction(tx); err != nil {
			errors = append(errors, fmt.Sprintf("tx %s: %v", tx.ID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback errors: %v", errors)
	}

	return nil
}

// logTransaction writes transaction state to log file
func (tl *TransactionLog) logTransaction(tx *Transaction) {
	data, err := json.Marshal(tx)
	if err != nil {
		return // Best effort logging
	}

	// Append to log file
	file, err := os.OpenFile(tl.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(string(data) + "\n")
	file.Sync()
}

// cleanupTransaction removes old transaction data
func (tl *TransactionLog) cleanupTransaction(txID string) {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()

	tx, exists := tl.transactions[txID]
	if !exists {
		return
	}

	// Clean up backup files for completed transactions
	if tx.Status == "completed" && tx.BackupPath != "" {
		os.Remove(tx.BackupPath)
	}

	delete(tl.transactions, txID)
}

// generateTransactionID creates a unique transaction ID
func generateTransactionID() string {
	suffix, _ := generateRandomSuffix()
	return fmt.Sprintf("tx_%d_%s", time.Now().UnixNano(), suffix[:8])
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Transaction represents a file operation transaction
type Transaction struct {
	ID         string    `json:"id"`
	TargetPath string    `json:"target_path"`
	TmpPath    string    `json:"tmp_path"`
	BackupPath string    `json:"backup_path"`
	Status     string    `json:"status"` // "pending", "completed", "failed", "rolled_back"
	Error      string    `json:"error,omitempty"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
}

// TransactionSummary provides a summary of transaction log state
type TransactionSummary struct {
	TotalTransactions      int `json:"total_transactions"`
	PendingTransactions    int `json:"pending_transactions"`
	CompletedTransactions  int `json:"completed_transactions"`
	FailedTransactions     int `json:"failed_transactions"`
	RolledBackTransactions int `json:"rolled_back_transactions"`
}

// GetSummary returns a summary of the transaction log
func (tl *TransactionLog) GetSummary() TransactionSummary {
	tl.mutex.RLock()
	defer tl.mutex.RUnlock()

	summary := TransactionSummary{}
	summary.TotalTransactions = len(tl.transactions)

	for _, tx := range tl.transactions {
		switch tx.Status {
		case "pending":
			summary.PendingTransactions++
		case "completed":
			summary.CompletedTransactions++
		case "failed":
			summary.FailedTransactions++
		case "rolled_back":
			summary.RolledBackTransactions++
		}
	}

	return summary
}
