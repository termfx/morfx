//go:build !windows

package core

import (
	"os"
	"syscall"
)

// isProcessAlive checks if a process with the given PID is alive on Unix-like systems
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists without affecting it
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
