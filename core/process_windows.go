//go:build windows

package core

import (
	"syscall"
	"unsafe"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess        = kernel32.NewProc("OpenProcess")
	procCloseHandle        = kernel32.NewProc("CloseHandle")
	procGetExitCodeProcess = kernel32.NewProc("GetExitCodeProcess")
)

const (
	PROCESS_QUERY_INFORMATION = 0x0400
	STILL_ACTIVE              = 259
)

// isProcessAlive checks if a process with the given PID is alive on Windows
func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	// Try to open the process with minimal permissions
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_QUERY_INFORMATION),
		uintptr(0), // bInheritHandle = FALSE
		uintptr(pid),
	)

	if handle == 0 {
		// Process doesn't exist or we can't access it
		return false
	}
	defer procCloseHandle.Call(handle)

	// Check if process is still active
	var exitCode uint32
	ret, _, _ := procGetExitCodeProcess.Call(
		handle,
		uintptr(unsafe.Pointer(&exitCode)),
	)

	if ret == 0 {
		// Failed to get exit code, assume dead
		return false
	}

	// Process is alive if exit code is STILL_ACTIVE
	return exitCode == STILL_ACTIVE
}
