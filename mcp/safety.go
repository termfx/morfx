package mcp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
)

// SafetyManager handles all safety-related operations
type SafetyManager struct {
	config     SafetyConfig
	fileLocks  map[string]*fileLock
	locksMutex sync.RWMutex
	txLog      *TransactionLog
}

// NewSafetyManager creates a new safety manager with the given config
func NewSafetyManager(config SafetyConfig) *SafetyManager {
	sm := &SafetyManager{
		config:    config,
		fileLocks: make(map[string]*fileLock),
	}

	if config.TransactionLog {
		sm.txLog = NewTransactionLog()
	}

	return sm
}

// ValidateOperation checks if an operation meets safety requirements
func (sm *SafetyManager) ValidateOperation(op *SafetyOperation) error {
	// Check file count limits
	if len(op.Files) > sm.config.MaxFiles {
		return NewMCPError(TooManyFiles,
			fmt.Sprintf("Operation exceeds file limit: %d > %d", len(op.Files), sm.config.MaxFiles),
			map[string]any{
				"requested": len(op.Files),
				"limit":     sm.config.MaxFiles,
			})
	}

	// Check individual file sizes and total size
	var totalSize int64
	for _, file := range op.Files {
		if file.Size > sm.config.MaxFileSize {
			return NewMCPError(FileTooLarge,
				fmt.Sprintf("File exceeds size limit: %s (%d bytes > %d bytes)",
					file.Path, file.Size, sm.config.MaxFileSize),
				map[string]any{
					"file":  file.Path,
					"size":  file.Size,
					"limit": sm.config.MaxFileSize,
				})
		}
		totalSize += file.Size
	}

	if totalSize > sm.config.MaxTotalSize {
		return NewMCPError(TotalSizeTooLarge,
			fmt.Sprintf("Total operation size exceeds limit: %d > %d", totalSize, sm.config.MaxTotalSize),
			map[string]any{
				"total": totalSize,
				"limit": sm.config.MaxTotalSize,
			})
	}

	// Validate confidence based on mode
	switch sm.config.ConfidenceMode {
	case "per_file":
		for _, file := range op.Files {
			if file.Confidence < sm.config.PerFileThreshold {
				return NewMCPError(PerFileConfidenceLow,
					fmt.Sprintf("File confidence too low: %s (%.3f < %.3f)",
						file.Path, file.Confidence, sm.config.PerFileThreshold),
					map[string]any{
						"file":       file.Path,
						"confidence": file.Confidence,
						"threshold":  sm.config.PerFileThreshold,
					})
			}
		}
	case "global":
		if op.GlobalConfidence < sm.config.GlobalThreshold {
			return NewMCPError(ConfidenceTooLow,
				fmt.Sprintf("Global confidence too low: %.3f < %.3f",
					op.GlobalConfidence, sm.config.GlobalThreshold))
		}
	case "both":
		// Check both per-file and global
		for _, file := range op.Files {
			if file.Confidence < sm.config.PerFileThreshold {
				return NewMCPError(PerFileConfidenceLow,
					fmt.Sprintf("File confidence too low: %s (%.3f < %.3f)",
						file.Path, file.Confidence, sm.config.PerFileThreshold))
			}
		}
		if op.GlobalConfidence < sm.config.GlobalThreshold {
			return NewMCPError(ConfidenceTooLow,
				fmt.Sprintf("Global confidence too low: %.3f < %.3f",
					op.GlobalConfidence, sm.config.GlobalThreshold))
		}
	}

	return nil
}

// ValidateFileIntegrity checks if files haven't been modified externally
func (sm *SafetyManager) ValidateFileIntegrity(files []FileIntegrityCheck) error {
	if !sm.config.ValidateFileHashes {
		return nil
	}

	for _, file := range files {
		currentHash, err := calculateFileHash(file.Path)
		if err != nil {
			return WrapError(FileSystemError,
				fmt.Sprintf("Failed to calculate hash for %s", file.Path), err)
		}

		if currentHash != file.ExpectedHash {
			return NewMCPError(FileModified,
				fmt.Sprintf("File was modified externally: %s", file.Path),
				map[string]any{
					"file":     file.Path,
					"expected": file.ExpectedHash,
					"actual":   currentHash,
				})
		}
	}

	return nil
}

// AtomicWrite performs an atomic write operation
func (sm *SafetyManager) AtomicWrite(path, content string) error {
	if !sm.config.AtomicWrites {
		// Fall back to regular write
		return os.WriteFile(path, []byte(content), 0o644)
	}

	// Create temporary file with random suffix
	suffix, err := generateRandomSuffix()
	if err != nil {
		return WrapError(AtomicWriteFailed, "Failed to generate random suffix", err)
	}

	tmpPath := path + ".tmp." + suffix

	// Create backup if requested
	var backupPath string
	if sm.config.CreateBackups {
		backupPath = path + sm.config.BackupSuffix
		if err := sm.createBackup(path, backupPath); err != nil {
			return WrapError(BackupFailed, "Failed to create backup", err)
		}
	}

	// Write to temporary file
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		sm.cleanupFailedWrite(tmpPath, backupPath)
		return WrapError(AtomicWriteFailed, "Failed to write temporary file", err)
	}

	// Fsync if requested
	if sm.config.UseFsync {
		if err := sm.syncFile(tmpPath); err != nil {
			sm.cleanupFailedWrite(tmpPath, backupPath)
			return WrapError(AtomicWriteFailed, "Failed to sync temporary file", err)
		}
	}

	// Log transaction
	if sm.txLog != nil {
		txID := sm.txLog.BeginTransaction(path, tmpPath, backupPath)
		defer sm.txLog.CompleteTransaction(txID)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		sm.cleanupFailedWrite(tmpPath, backupPath)
		return WrapError(AtomicWriteFailed, "Failed to rename temporary file", err)
	}

	// Fsync directory if requested
	if sm.config.UseFsync {
		if err := sm.syncDir(filepath.Dir(path)); err != nil {
			// File is already written, this is just a warning
			fmt.Fprintf(os.Stderr, "Warning: Failed to sync directory: %v\n", err)
		}
	}

	return nil
}

// LockFile acquires an exclusive lock on a file
func (sm *SafetyManager) LockFile(path string) (*FileLock, error) {
	if path == "" {
		return nil, NewMCPError(InvalidParams, "File path cannot be empty")
	}

	if !sm.config.FileLocking {
		return &FileLock{path: path, manager: sm}, nil // No-op lock
	}

	sm.locksMutex.Lock()
	defer sm.locksMutex.Unlock()

	// Check if already locked
	if _, exists := sm.fileLocks[path]; exists {
		return nil, NewMCPError(FileLocked,
			fmt.Sprintf("File is already locked: %s", path))
	}

	// Try to acquire OS-level lock
	lock, err := sm.acquireOSLock(path)
	if err != nil {
		return nil, err
	}

	sm.fileLocks[path] = lock
	return &FileLock{path: path, manager: sm, osLock: lock}, nil
}

// ReleaseLock releases a file lock
func (sm *SafetyManager) ReleaseLock(path string) error {
	if !sm.config.FileLocking {
		return nil
	}

	sm.locksMutex.Lock()
	defer sm.locksMutex.Unlock()

	lock, exists := sm.fileLocks[path]
	if !exists {
		return nil // Already released
	}

	if err := lock.release(); err != nil {
		return WrapError(FileSystemError, "Failed to release file lock", err)
	}

	delete(sm.fileLocks, path)
	return nil
}

// acquireOSLock tries to acquire an OS-level file lock with timeout
func (sm *SafetyManager) acquireOSLock(path string) (*fileLock, error) {
	lockPath := path + ".lock"

	deadline := time.Now().Add(sm.config.LockTimeout)
	for time.Now().Before(deadline) {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			// Successfully created lock file
			lock := &fileLock{
				path:     lockPath,
				file:     file,
				acquired: time.Now(),
			}

			// Write PID to lock file
			fmt.Fprintf(file, "%d\n", os.Getpid())
			file.Sync()

			return lock, nil
		}

		// Check if lock is stale
		if sm.isLockStale(lockPath) {
			os.Remove(lockPath) // Remove stale lock
			continue
		}

		// Wait a bit before retrying
		time.Sleep(100 * time.Millisecond)
	}

	return nil, NewMCPError(LockTimeout,
		fmt.Sprintf("Could not acquire lock for %s within %v", path, sm.config.LockTimeout))
}

// isLockStale checks if a lock file is stale (process no longer exists)
func (sm *SafetyManager) isLockStale(lockPath string) bool {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return true // Can't read = stale
	}

	var pid int
	if n, err := fmt.Sscanf(string(content), "%d", &pid); err != nil || n != 1 {
		return true // Invalid format = stale
	}

	return !isProcessAlive(pid)
}

// Helper functions

func (sm *SafetyManager) createBackup(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to backup
		}
		return err
	}
	return os.WriteFile(dst, content, 0o644)
}

func (sm *SafetyManager) cleanupFailedWrite(tmpPath, backupPath string) {
	os.Remove(tmpPath)
	if backupPath != "" {
		os.Remove(backupPath)
	}
}

func (sm *SafetyManager) syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func (sm *SafetyManager) syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// calculateFileHash computes SHA256 hash of a file
func calculateFileHash(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// generateRandomSuffix creates a random suffix for temporary files
func generateRandomSuffix() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Data structures

type SafetyOperation struct {
	Files            []SafetyFile `json:"files"`
	GlobalConfidence float64      `json:"global_confidence"`
}

type SafetyFile struct {
	Path       string  `json:"path"`
	Size       int64   `json:"size"`
	Confidence float64 `json:"confidence"`
}

type FileIntegrityCheck struct {
	Path         string `json:"path"`
	ExpectedHash string `json:"expected_hash"`
}

type FileLock struct {
	path    string
	manager *SafetyManager
	osLock  *fileLock
}

func (fl *FileLock) Release() error {
	if fl.manager != nil {
		return fl.manager.ReleaseLock(fl.path)
	}
	return nil
}

type fileLock struct {
	path     string
	file     *os.File
	acquired time.Time
}

func (fl *fileLock) release() error {
	if fl.file != nil {
		fl.file.Close()
		return os.Remove(fl.path)
	}
	return nil
}

// isProcessAlive checks if a process exists cross-platform
func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, FindProcess always succeeds, but we can test if process exists
		// by trying to get its exit code
		return process.Signal(os.Interrupt) == nil
	}

	// Unix-like: use signal 0 to test existence
	return process.Signal(syscall.Signal(0)) == nil
}
