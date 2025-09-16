//go:build integration

package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestProcessDetection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var cmd *exec.Cmd

	// Start a long-running process based on OS
	if runtime.GOOS == "windows" {
		// Windows: use timeout command to sleep
		cmd = exec.Command("timeout", "5")
	} else {
		// Unix: use sleep
		cmd = exec.Command("sleep", "5")
	}

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Test that we can detect the live process
	if !isProcessAlive(cmd.Process.Pid) {
		t.Errorf("Process %d should be detected as alive", cmd.Process.Pid)
	}

	// Kill the process
	err = cmd.Process.Kill()
	if err != nil {
		t.Fatalf("Failed to kill test process: %v", err)
	}

	// Wait a moment for the process to die
	cmd.Wait()
	time.Sleep(100 * time.Millisecond)

	// Test that we can detect the dead process
	if isProcessAlive(cmd.Process.Pid) {
		t.Errorf("Process %d should be detected as dead after kill", cmd.Process.Pid)
	}
}

func TestAtomicWriter_LockStaleDetection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	lockPath := testFile + ".lock"

	config := DefaultAtomicConfig()
	config.LockTimeout = 200 * time.Millisecond
	aw := NewAtomicWriter(config)

	// Create a lock file with a dead process PID
	deadPID := 999999 // Very likely to be dead
	err := os.WriteFile(lockPath, []byte(fmt.Sprintf("%d", deadPID)), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	// This should succeed because the lock is stale
	err = aw.WriteFile(testFile, "test content")
	if err != nil {
		t.Errorf("Write should succeed with stale lock: %v", err)
	}

	// Verify content was written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected 'test content', got %q", string(content))
	}
}
