# Morfx Advanced Safety Features

## Overview

Morfx includes comprehensive safety features to ensure reliable and recoverable code transformations. These features protect against data loss, corruption, and concurrent access issues.

## Safety Architecture

### Phase 3 Implementation - Advanced Safety ✅

The safety system consists of three main components:

1. **AtomicWriter** - Handles atomic file operations with locking
2. **TransactionManager** - Provides transaction logging and rollback capabilities  
3. **Enhanced FileProcessor** - Integrates safety features with existing functionality

## Key Features

### 🔒 Atomic File Operations

```go
// Configure atomic writing
config := core.AtomicWriteConfig{
    UseFsync:       true,  // Force fsync for durability
    LockTimeout:    10 * time.Second,
    TempSuffix:     ".morfx.tmp",
    BackupOriginal: true,
}

atomicWriter := core.NewAtomicWriter(config)
err := atomicWriter.WriteFile("/path/to/file.go", newContent)
```

**Benefits:**
- Prevents partial writes and corruption
- Uses temporary files + atomic rename
- Optional fsync for durability guarantees
- Concurrent access protection via file locking

### 📋 Transaction Logging

```go
// Begin transaction
tx, err := txManager.BeginTransaction("Refactor API endpoints")

// Add operations
op, err := txManager.AddOperation("modify", "/path/to/file.go")

// Complete transaction
err = txManager.CommitTransaction()
// Note: Rollback happens automatically on errors
```

**Benefits:**
- Complete audit trail of all changes  
- Atomic batch operations
- Automatic error recovery
- Automatic backup creation

### 🔄 Automatic Safety Rollback

```go
// Automatic rollback on transformation errors
result, err := processor.TransformFiles(ctx, transformOp)
// If errors occur during batch operations, changes are automatically reverted

// Automatic rollback on low confidence
if result.Confidence.Score < 0.3 {
    // Auto rollback triggered to prevent dangerous transformations
}
```

**Benefits:**
- Protection against corruption and failed operations
- Automatic recovery from batch transformation errors
- Confidence-based safety checks prevent risky changes

### 📊 Confidence-Based Validation

```go
result, err := processor.TransformFiles(ctx, transformOp)

// Check overall confidence  
if result.Confidence.Score < 0.8 {
    log.Printf("Low confidence transformation: %.2f", result.Confidence.Score)
    // Note: Use dry-run mode for risky transformations
}

// Check individual file confidence
for _, file := range result.Files {
    if file.Confidence.Score < 0.5 {
        log.Printf("Very low confidence for %s: %.2f", 
            file.FilePath, file.Confidence.Score)
    }
}
```

**Benefits:**
- Automatic risk assessment
- Prevents dangerous transformations
- Detailed confidence factors and explanations

## Configuration Options

### Safety Levels

**High Performance (Low Safety)**
```go
config := core.AtomicWriteConfig{
    UseFsync:       false, // Skip fsync
    LockTimeout:    1 * time.Second,
    BackupOriginal: false, // Skip backup
}
processor := core.NewFileProcessorWithSafety(registry, false, config)
```

**Balanced (Default)**
```go
processor := core.NewFileProcessor(registry) // Uses DefaultAtomicConfig
```

**Maximum Safety**
```go
config := core.AtomicWriteConfig{
    UseFsync:       true,  // Force fsync
    LockTimeout:    30 * time.Second,
    BackupOriginal: true,  // Always backup
}
processor := core.NewFileProcessorWithSafety(registry, true, config)
```

### Runtime Configuration

```go
// Enable/disable safety at runtime
processor.EnableSafety(true)

// Check current status
if processor.IsSafetyEnabled() {
    log.Println("Safety features are active")
}

// Cleanup on shutdown (important!)
defer processor.Cleanup()
```

## File System Layout

When safety features are enabled, Morfx creates this structure:

```
.morfx/
├── transactions/           # Transaction logs
│   ├── tx_1234567890_5678.json
│   └── tx_1234567891_9012.json
├── backups/               # Automatic backups
│   ├── .morfx-backup-file.go-tx_123-20240914-153045
│   └── .morfx-backup-api.go-tx_124-20240914-153046
└── locks/                 # File locks (temporary)
    ├── file.go.lock
    └── api.go.lock
```

## Transaction Log Format

```json
{
  "id": "tx_1694712345_1234",
  "started": "2024-09-14T15:30:45Z",
  "completed": "2024-09-14T15:30:47Z",
  "status": "committed",
  "description": "Refactor authentication functions",
  "operations": [
    {
      "type": "modify",
      "file_path": "/path/to/auth.go",
      "backup_path": ".morfx-backup-auth.go-tx_123-20240914-153045",
      "checksum": "abc123...",
      "timestamp": "2024-09-14T15:30:45Z",
      "completed": true
    }
  ]
}
```

## Error Handling

### Concurrent Access Protection

```go
// Automatic retry with exponential backoff
err := atomicWriter.WriteFile("/path/to/file.go", content)
if err != nil {
    // Handle lock timeout or other errors
    log.Printf("Write failed: %v", err)
}
```

### Transaction Recovery

```go
// Transactions are automatically handled by the safety system
// Failed operations trigger automatic rollback
result, err := processor.TransformFiles(ctx, transformOp)
if err != nil {
    // System automatically reverted any partial changes
    log.Printf("Transform failed, changes reverted: %v", err)
}
```

### Validation Failures

```go
// Check if transformation passed validation
err = processor.ValidateChanges(result.Files)
if err != nil {
    log.Printf("Validation failed: %v", err)
    // Low confidence transformations are automatically prevented
}
```

## Best Practices

1. **Always call Cleanup()** on shutdown to release locks
2. **Monitor confidence scores** and set appropriate thresholds  
3. **Use dry-run mode** for risky transformations first
4. **Choose safety level** based on your use case:
   - Development: Balanced mode
   - CI/CD: High performance mode
   - Production: Maximum safety mode
5. **Test transformations** on small file sets before large batches

## Performance Considerations

| Feature | Performance Impact | Safety Benefit |
|---------|-------------------|----------------|
| File Locking | Low | Prevents corruption |
| Fsync | High | Guarantees durability |
| Backup Creation | Medium | Automatic recovery |
| Transaction Logging | Low | Audit trail |
| Checksum Generation | Low | Integrity verification |

## Configuration Examples

### Basic Safety Configuration
```go
// Default configuration with balanced safety/performance
processor := core.NewFileProcessor(registry)
defer processor.Cleanup()
```

### High Performance Mode
```go
config := core.AtomicWriteConfig{
    UseFsync:       false, // Skip fsync for speed
    LockTimeout:    1 * time.Second,
    BackupOriginal: false, // Skip backup for speed
}
processor := core.NewFileProcessorWithSafety(registry, false, config)
```

### Maximum Safety Mode
```go
config := core.AtomicWriteConfig{
    UseFsync:       true,  // Force fsync for durability
    LockTimeout:    30 * time.Second,
    BackupOriginal: true,  // Always backup
}
processor := core.NewFileProcessorWithSafety(registry, true, config)
```

## Migration Guide

### From Basic FileProcessor

```go
// Before
processor := core.NewFileProcessor(registry)

// After  
processor := core.NewFileProcessor(registry) // Safety enabled by default
// or
processor := core.NewFileProcessorWithSafety(registry, true, customConfig)
defer processor.Cleanup()
```

### Handling Results

```go
// Before
result, err := processor.TransformFiles(ctx, op)

// After
result, err := processor.TransformFiles(ctx, op)
if err != nil {
    return err // Automatic rollback already handled
}

// New: Check confidence scores for validation
if result.Confidence.Score < 0.8 {
    log.Printf("Low confidence transformation: %.2f", result.Confidence.Score)
    // Consider running with dry-run first
}
```
