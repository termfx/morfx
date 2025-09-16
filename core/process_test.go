package core

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsProcessAlive_CurrentProcess(t *testing.T) {
	// Current process should always be alive
	currentPID := os.Getpid()
	if !isProcessAlive(currentPID) {
		t.Errorf("Current process (PID %d) should be alive", currentPID)
	}
}

func TestIsProcessAlive_InvalidPID(t *testing.T) {
	// Invalid PIDs should return false
	testCases := []int{-1, 0, 999999999}

	for _, pid := range testCases {
		if isProcessAlive(pid) {
			t.Errorf("Invalid PID %d should be reported as dead", pid)
		}
	}
}

func TestIsProcessAlive_DeadProcess(t *testing.T) {
	// We can't easily test a dead process in unit tests without race conditions
	// This would be better as an integration test
	t.Skip("Dead process testing requires integration test setup")
}

func TestAtomicWriter_IsLockStale_CrossPlatform(t *testing.T) {
	tempDir := t.TempDir()
	config := DefaultAtomicConfig()
	aw := NewAtomicWriter(config)

	lockPath := filepath.Join(tempDir, "test.lock")

	t.Run("NonexistentLock", func(t *testing.T) {
		if !aw.isLockStale(lockPath) {
			t.Error("Nonexistent lock should be considered stale")
		}
	})

	t.Run("InvalidPIDFormat", func(t *testing.T) {
		err := os.WriteFile(lockPath, []byte("not-a-number"), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		if !aw.isLockStale(lockPath) {
			t.Error("Lock with invalid PID format should be considered stale")
		}
	})

	t.Run("CurrentProcessPID", func(t *testing.T) {
		currentPID := os.Getpid()
		err := os.WriteFile(lockPath, fmt.Appendf(nil, "%d", currentPID), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		if aw.isLockStale(lockPath) {
			t.Error("Lock with current process PID should not be considered stale")
		}
	})

	t.Run("InvalidPID", func(t *testing.T) {
		err := os.WriteFile(lockPath, []byte("999999999"), 0o644)
		if err != nil {
			t.Fatal(err)
		}

		if !aw.isLockStale(lockPath) {
			t.Error("Lock with invalid PID should be considered stale")
		}
	})
}

func TestAtomicWriter_LockTimeout_CrossPlatform(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create a lock that simulates a live process
	currentPID := os.Getpid()
	lockPath := testFile + ".lock"
	err := os.WriteFile(lockPath, fmt.Appendf(nil, "%d", currentPID), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	config := DefaultAtomicConfig()
	config.LockTimeout = 100 * time.Millisecond // Short timeout for test
	aw := NewAtomicWriter(config)

	// This should timeout because we're simulating our own process holding the lock
	err = aw.WriteFile(testFile, "test content")
	if err == nil {
		t.Error("Expected timeout error when lock is held by live process")
	}

	// Clean up
	os.Remove(lockPath)
}

// BenchmarkIsProcessAlive tests performance on different platforms
func BenchmarkIsProcessAlive(b *testing.B) {
	currentPID := os.Getpid()

	b.Run("AliveProcess", func(b *testing.B) {
		for b.Loop() {
			isProcessAlive(currentPID)
		}
	})

	b.Run("DeadProcess", func(b *testing.B) {
		for b.Loop() {
			isProcessAlive(999999999) // Likely dead PID
		}
	})
}
