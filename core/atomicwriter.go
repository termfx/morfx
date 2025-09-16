package core

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// FileLock represents a file lock for concurrent access control
type FileLock struct {
	file   *os.File
	path   string
	locked bool
	mu     sync.Mutex
}

// AtomicWriteConfig controls atomic writing behavior
type AtomicWriteConfig struct {
	UseFsync       bool          // Force fsync for durability
	LockTimeout    time.Duration // Max time to wait for file lock
	TempSuffix     string        // Suffix for temporary files
	BackupOriginal bool          // Create backup before writing
}

// DefaultAtomicConfig provides sensible defaults
func DefaultAtomicConfig() AtomicWriteConfig {
	return AtomicWriteConfig{
		UseFsync:       false, // Performance over safety by default
		LockTimeout:    5 * time.Second,
		TempSuffix:     ".morfx.tmp",
		BackupOriginal: true,
	}
}

// AtomicWriter handles atomic file operations with locking
type AtomicWriter struct {
	config AtomicWriteConfig
	locks  map[string]*FileLock
	mu     sync.RWMutex
}

// NewAtomicWriter creates a new atomic writer
func NewAtomicWriter(config AtomicWriteConfig) *AtomicWriter {
	return &AtomicWriter{
		config: config,
		locks:  make(map[string]*FileLock),
	}
}

// WriteFile atomically writes content to file with optional locking
func (aw *AtomicWriter) WriteFile(path, content string) error {
	// Acquire exclusive lock
	if err := aw.acquireLock(path); err != nil {
		return fmt.Errorf("failed to acquire lock for %s: %w", path, err)
	}
	defer aw.releaseLock(path)

	// Get original file info
	originalInfo, err := os.Stat(path)
	var fileMode os.FileMode = 0o644
	if err == nil {
		fileMode = originalInfo.Mode()
	}

	// Create backup if requested and file exists
	var backupPath string
	if aw.config.BackupOriginal && err == nil {
		backupPath = path + ".bak"
		if err := aw.createBackup(path, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Write to temporary file first
	tempPath := path + aw.config.TempSuffix
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write content
	_, err = tempFile.WriteString(content)
	if err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Force sync if requested
	if aw.config.UseFsync {
		if err := tempFile.Sync(); err != nil {
			tempFile.Close()
			os.Remove(tempPath)
			return fmt.Errorf("failed to sync: %w", err)
		}
	}

	tempFile.Close()

	// Atomic rename (the critical atomic operation)
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to atomic rename: %w", err)
	}

	return nil
}

// acquireLock gets an exclusive file lock
func (aw *AtomicWriter) acquireLock(path string) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	// Check if we already have a lock for this file
	if _, exists := aw.locks[path]; exists {
		return nil // Already locked
	}

	// Create lock file path
	lockPath := path + ".lock"

	// Try to acquire lock with timeout
	deadline := time.Now().Add(aw.config.LockTimeout)
	for time.Now().Before(deadline) {
		lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			// Lock acquired
			lock := &FileLock{
				file:   lockFile,
				path:   lockPath,
				locked: true,
			}
			aw.locks[path] = lock

			// Write PID to lock file for debugging
			fmt.Fprintf(lockFile, "%d\n", os.Getpid())
			lockFile.Sync()

			return nil
		}

		// Check if it's because file exists (lock held by another process)
		if os.IsExist(err) {
			// Check if lock is stale
			if aw.isLockStale(lockPath) {
				os.Remove(lockPath) // Remove stale lock
				continue
			}

			// Wait a bit and retry
			time.Sleep(100 * time.Millisecond)
			continue
		}

		return fmt.Errorf("failed to create lock file: %w", err)
	}

	return fmt.Errorf("timeout waiting for lock on %s", path)
}

// releaseLock releases the file lock
func (aw *AtomicWriter) releaseLock(path string) error {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	lock, exists := aw.locks[path]
	if !exists {
		return nil // Already released
	}

	lock.mu.Lock()
	defer lock.mu.Unlock()

	if lock.locked {
		lock.file.Close()
		os.Remove(lock.path)
		lock.locked = false
	}

	delete(aw.locks, path)
	return nil
}

// isLockStale checks if a lock file is from a dead process (cross-platform)
func (aw *AtomicWriter) isLockStale(lockPath string) bool {
	content, err := os.ReadFile(lockPath)
	if err != nil {
		return true // Can't read, assume stale
	}

	var pid int
	if _, err := fmt.Sscanf(string(content), "%d", &pid); err != nil {
		return true // Invalid format, assume stale
	}

	// Cross-platform process check
	return !isProcessAlive(pid)
}

// createBackup creates a backup copy with timestamp
func (aw *AtomicWriter) createBackup(originalPath, backupPath string) error {
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return err
	}

	// Add timestamp to backup
	timestamp := time.Now().Format("20060102-150405")
	backupPath = fmt.Sprintf("%s.%s", backupPath, timestamp)

	return os.WriteFile(backupPath, content, 0o644)
}

// Cleanup removes all locks (call on shutdown)
func (aw *AtomicWriter) Cleanup() {
	aw.mu.Lock()
	defer aw.mu.Unlock()

	for path := range aw.locks {
		aw.releaseLock(path)
	}
}
