package core

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/oxhq/morfx/internal/securefs"
)

// FileLock represents a file lock for concurrent access control
type FileLock struct {
	file   *os.File
	path   string
	locked bool
	mu     sync.Mutex
	cond   *sync.Cond
	refCnt int
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
func (aw *AtomicWriter) WriteFile(path, content string) (err error) {
	// Acquire exclusive lock
	if err := aw.acquireLock(path); err != nil {
		return fmt.Errorf("failed to acquire lock for %s: %w", path, err)
	}
	defer func() {
		if releaseErr := aw.releaseLock(path); releaseErr != nil && err == nil {
			err = releaseErr
		}
	}()

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
	tempFile, err := securefs.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write content
	_, err = tempFile.WriteString(content)
	if err != nil {
		securefs.CloseBestEffort(tempFile)
		securefs.RemoveBestEffort(tempPath)
		return fmt.Errorf("failed to write content: %w", err)
	}

	// Force sync if requested
	if aw.config.UseFsync {
		if err := tempFile.Sync(); err != nil {
			securefs.CloseBestEffort(tempFile)
			securefs.RemoveBestEffort(tempPath)
			return fmt.Errorf("failed to sync: %w", err)
		}
	}

	if err := tempFile.Close(); err != nil {
		securefs.RemoveBestEffort(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename (the critical atomic operation)
	if err := os.Rename(tempPath, path); err != nil {
		securefs.RemoveBestEffort(tempPath)
		return fmt.Errorf("failed to atomic rename: %w", err)
	}

	return nil
}

// acquireLock gets an exclusive file lock
func (aw *AtomicWriter) acquireLock(path string) error {
	lockPath := path + ".lock"

	aw.mu.Lock()
	lock, exists := aw.locks[path]
	if !exists {
		lock = &FileLock{}
		aw.locks[path] = lock
	}
	if lock.cond == nil {
		lock.cond = sync.NewCond(&lock.mu)
	}
	lock.path = lockPath
	lock.refCnt++
	aw.mu.Unlock()

	// Wait for in-process writers to finish
	lock.mu.Lock()
	for lock.locked {
		lock.cond.Wait()
	}
	lock.mu.Unlock()

	deadline := time.Now().Add(aw.config.LockTimeout)
	for {
		lockFile, err := securefs.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			lock.mu.Lock()
			lock.file = lockFile
			lock.locked = true
			lock.mu.Unlock()

			// Write PID to lock file for debugging
			if _, err := fmt.Fprintf(lockFile, "%d\n", os.Getpid()); err != nil {
				if releaseErr := aw.releaseLock(path); releaseErr != nil {
					return fmt.Errorf("failed to write lock file: %w; failed to release lock: %w", err, releaseErr)
				}
				return fmt.Errorf("failed to write lock file: %w", err)
			}
			if err := lockFile.Sync(); err != nil {
				if releaseErr := aw.releaseLock(path); releaseErr != nil {
					return fmt.Errorf("failed to sync lock file: %w; failed to release lock: %w", err, releaseErr)
				}
				return fmt.Errorf("failed to sync lock file: %w", err)
			}

			return nil
		}

		if os.IsExist(err) {
			if aw.isLockStale(lockPath) {
				securefs.RemoveBestEffort(lockPath)
				continue
			}
			if time.Now().After(deadline) {
				aw.decrementRefCount(path, lock)
				return fmt.Errorf("timeout waiting for lock on %s", path)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		aw.decrementRefCount(path, lock)
		return fmt.Errorf("failed to create lock file: %w", err)
	}
}

// releaseLock releases the file lock
func (aw *AtomicWriter) releaseLock(path string) error {
	aw.mu.RLock()
	lock, exists := aw.locks[path]
	aw.mu.RUnlock()
	if !exists {
		return nil // Already released
	}

	lock.mu.Lock()
	var releaseErr error
	if lock.locked {
		var closeErr error
		if lock.file != nil {
			closeErr = lock.file.Close()
		}
		removeErr := os.Remove(lock.path)
		lock.locked = false
		lock.file = nil
		lock.cond.Broadcast()
		if closeErr != nil {
			releaseErr = fmt.Errorf("failed to close lock file: %w", closeErr)
		} else if removeErr != nil && !os.IsNotExist(removeErr) {
			releaseErr = fmt.Errorf("failed to remove lock file: %w", removeErr)
		}
	}
	lock.refCnt--
	remove := lock.refCnt == 0
	lock.mu.Unlock()

	if remove {
		aw.mu.Lock()
		if l, ok := aw.locks[path]; ok {
			l.mu.Lock()
			if l.refCnt == 0 && !l.locked {
				delete(aw.locks, path)
			}
			l.mu.Unlock()
		}
		aw.mu.Unlock()
	}
	return releaseErr
}

// isLockStale checks if a lock file is from a dead process (cross-platform)
func (aw *AtomicWriter) isLockStale(lockPath string) bool {
	content, err := securefs.ReadFile(lockPath)
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
	info, err := os.Stat(originalPath)
	if err != nil {
		return err
	}

	content, err := securefs.ReadFile(originalPath)
	if err != nil {
		return err
	}

	// Add timestamp to backup
	timestamp := time.Now().Format("20060102-150405")
	backupPath = fmt.Sprintf("%s.%s", backupPath, timestamp)

	perm := info.Mode().Perm()
	if perm == 0 {
		perm = 0o644
	}

	if err := securefs.WriteFile(backupPath, content, perm); err != nil {
		return err
	}
	return os.Chmod(backupPath, perm)
}

// Cleanup removes all locks (call on shutdown)
func (aw *AtomicWriter) Cleanup() {
	aw.mu.RLock()
	paths := make([]string, 0, len(aw.locks))
	for path := range aw.locks {
		paths = append(paths, path)
	}
	aw.mu.RUnlock()

	for _, path := range paths {
		securefs.IgnoreError(aw.releaseLock(path))
	}
}

// decrementRefCount adjusts the reference count when acquisition fails.
func (aw *AtomicWriter) decrementRefCount(path string, lock *FileLock) {
	lock.mu.Lock()
	if lock.refCnt > 0 {
		lock.refCnt--
	}
	remove := lock.refCnt == 0 && !lock.locked
	lock.mu.Unlock()

	if remove {
		aw.mu.Lock()
		if l, ok := aw.locks[path]; ok {
			l.mu.Lock()
			if l.refCnt == 0 && !l.locked {
				delete(aw.locks, path)
			}
			l.mu.Unlock()
		}
		aw.mu.Unlock()
	}
}
