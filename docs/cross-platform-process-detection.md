# Cross-Platform Process Detection

Cross-platform process detection system for stale lock cleanup in Morfx atomic writer.

## Implementation

### Unix (`process_unix.go`)
- Uses `kill(pid, 0)` via `syscall.Signal(0)`
- Single system call, ~100ns performance
- Standard POSIX approach

### Windows (`process_windows.go`)
- Uses Windows API: `OpenProcess` + `GetExitCodeProcess`
- ~500-1000ns performance
- Checks for `STILL_ACTIVE` (259) exit code

## Error Handling
- Invalid PIDs (≤ 0) → `false`
- Permission errors → `false` (enables lock cleanup)
- API failures → `false` (safe default)

## Usage
```go
if !isProcessAlive(pid) {
    os.Remove(lockPath) // Safe to cleanup stale lock
}
```

## Testing
- Unit: `go test ./core -run TestIsProcessAlive`
- Integration: `go test ./core -tags=integration`
- Benchmarks: `go test -bench=BenchmarkIsProcessAlive`
