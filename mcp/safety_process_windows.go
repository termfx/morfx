//go:build windows

package mcp

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
	processQueryInformation = 0x0400
	stillActive             = 259
)

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	handle, _, _ := procOpenProcess.Call(
		uintptr(processQueryInformation),
		uintptr(0),
		uintptr(uint32(pid)),
	)
	if handle == 0 {
		return false
	}
	defer procCloseHandle.Call(handle)

	var exitCode uint32
	ret, _, _ := procGetExitCodeProcess.Call(
		handle,
		uintptr(unsafe.Pointer(&exitCode)),
	)
	if ret == 0 {
		return false
	}

	return exitCode == stillActive
}
