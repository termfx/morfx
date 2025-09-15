package core

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"
	"time"
)

// ProviderRegistry interface for provider lookup
type ProviderRegistry interface {
	Get(language string) (Provider, bool)
}

// Provider interface for language-specific operations
type Provider interface {
	Language() string
	Query(source string, query AgentQuery) QueryResult
	Transform(source string, op TransformOp) TransformResult
}

// FileProcessor handles file-based transformations using providers
type FileProcessor struct {
	walker        *FileWalker
	providers     ProviderRegistry
	workers       int
	atomicWriter  *AtomicWriter
	txManager     *TransactionManager
	safetyEnabled bool
}

// NewFileProcessor creates a new file processor
func NewFileProcessor(providerRegistry ProviderRegistry) *FileProcessor {
	atomicConfig := DefaultAtomicConfig()
	atomicConfig.UseFsync = false // Performance over safety by default

	atomicWriter := NewAtomicWriter(atomicConfig)
	txManager := NewTransactionManager(".morfx/transactions", atomicWriter)

	return &FileProcessor{
		walker:        NewFileWalker(),
		providers:     providerRegistry,
		workers:       8, // Parallel file processing workers
		atomicWriter:  atomicWriter,
		txManager:     txManager,
		safetyEnabled: true,
	}
}

// NewFileProcessorWithSafety creates a processor with configurable safety settings
func NewFileProcessorWithSafety(
	providerRegistry ProviderRegistry,
	safetyEnabled bool,
	atomicConfig AtomicWriteConfig,
) *FileProcessor {
	atomicWriter := NewAtomicWriter(atomicConfig)
	txManager := NewTransactionManager(".morfx/transactions", atomicWriter)

	return &FileProcessor{
		walker:        NewFileWalker(),
		providers:     providerRegistry,
		workers:       8,
		atomicWriter:  atomicWriter,
		txManager:     txManager,
		safetyEnabled: safetyEnabled,
	}
}

// QueryFiles searches for code elements across multiple files
func (fp *FileProcessor) QueryFiles(ctx context.Context, scope FileScope, query AgentQuery) ([]FileMatch, error) {
	start := time.Now()

	// Discover files
	results, err := fp.walker.Walk(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to walk files: %w", err)
	}

	// Process files in parallel
	matches := make(chan []FileMatch, fp.workers)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < fp.workers; i++ {
		wg.Add(1)
		go fp.queryWorker(ctx, results, query, matches, &wg)
	}

	// Collect results
	var allMatches []FileMatch
	go func() {
		for fileMatches := range matches {
			allMatches = append(allMatches, fileMatches...)
		}
	}()

	// Wait for completion
	wg.Wait()
	close(matches)

	// Add timing metadata
	duration := time.Since(start)
	fmt.Printf("Query completed in %v, found %d matches across files\n", duration, len(allMatches))

	return allMatches, nil
}

// TransformFiles applies transformations across multiple files
func (fp *FileProcessor) TransformFiles(ctx context.Context, op FileTransformOp) (*FileTransformResult, error) {
	start := time.Now()

	// Start transaction if safety enabled
	var tx *TransactionLog
	if fp.safetyEnabled && !op.DryRun {
		var err error
		tx, err = fp.txManager.BeginTransaction(fmt.Sprintf("Transform files: %s", op.TransformOp.Target.Type))
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Ensure cleanup on any error
		defer func() {
			if tx != nil && fp.txManager.currentTx != nil {
				fp.txManager.RollbackTransaction()
			}
		}()
	}

	// Discover files
	walkResults, err := fp.walker.Walk(ctx, op.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to walk files: %w", err)
	}

	// Collect file paths
	var filePaths []WalkResult
	for result := range walkResults {
		if result.Error == nil {
			filePaths = append(filePaths, result)
		}
	}

	scanDuration := time.Since(start)
	transformStart := time.Now()

	// Process files in parallel
	resultChan := make(chan FileTransformDetail, len(filePaths))
	var wg sync.WaitGroup

	// Create semaphore for controlled parallelism
	semaphore := make(chan struct{}, fp.workers)

	for _, walkResult := range filePaths {
		wg.Add(1)
		go func(wr WalkResult) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			detail := fp.transformFile(wr, op, tx)
			resultChan <- detail
		}(walkResult)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var details []FileTransformDetail
	var totalMatches int
	var filesModified int
	var hasErrors bool

	for detail := range resultChan {
		details = append(details, detail)
		totalMatches += detail.MatchCount
		if detail.Modified {
			filesModified++
		}
		if detail.Error != "" {
			hasErrors = true
		}
	}

	transformDuration := time.Since(transformStart)

	// Handle transaction completion
	if fp.safetyEnabled && !op.DryRun && tx != nil {
		if hasErrors {
			if err := fp.txManager.RollbackTransaction(); err != nil {
				return nil, fmt.Errorf("failed to rollback transaction: %w", err)
			}
		} else {
			if err := fp.txManager.CommitTransaction(); err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}
			tx = nil // Prevent rollback in defer
		}
	}

	// Calculate overall confidence
	overallConfidence := fp.calculateOverallConfidence(details)

	return &FileTransformResult{
		FilesScanned:      len(filePaths),
		FilesModified:     filesModified,
		TotalMatches:      totalMatches,
		ScanDuration:      scanDuration.Milliseconds(),
		TransformDuration: transformDuration.Milliseconds(),
		Files:             details,
		Confidence:        overallConfidence,
		TransactionID: func() string {
			if tx != nil {
				return tx.ID
			}
			return ""
		}(),
	}, nil
}

// queryWorker processes files for queries in parallel
func (fp *FileProcessor) queryWorker(
	ctx context.Context,
	results <-chan WalkResult,
	query AgentQuery,
	matches chan<- []FileMatch,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-results:
			if !ok {
				return
			}

			if result.Error != nil {
				continue
			}

			fileMatches := fp.queryFile(result, query)
			if len(fileMatches) > 0 {
				matches <- fileMatches
			}
		}
	}
}

// queryFile searches for matches in a single file
func (fp *FileProcessor) queryFile(walkResult WalkResult, query AgentQuery) []FileMatch {
	// Read file content
	content, err := os.ReadFile(walkResult.Path)
	if err != nil {
		return nil
	}

	// Get provider for language
	provider, exists := fp.providers.Get(walkResult.Language)
	if !exists {
		return nil // Skip unsupported languages
	}

	// Execute query
	result := provider.Query(string(content), query)
	if result.Error != nil {
		return nil
	}

	// Convert to FileMatch
	var fileMatches []FileMatch
	for _, match := range result.Matches {
		fileMatch := FileMatch{
			Match:    match,
			FilePath: walkResult.Path,
			FileSize: walkResult.Info.Size(),
			ModTime:  walkResult.Info.ModTime().Unix(),
			Language: walkResult.Language,
		}
		// Update location to include file
		fileMatch.Location.File = walkResult.Path
		fileMatches = append(fileMatches, fileMatch)
	}

	return fileMatches
}

// transformFile applies transformation to a single file
func (fp *FileProcessor) transformFile(
	walkResult WalkResult,
	op FileTransformOp,
	tx *TransactionLog,
) FileTransformDetail {
	detail := FileTransformDetail{
		FilePath:     walkResult.Path,
		Language:     walkResult.Language,
		OriginalSize: walkResult.Info.Size(),
	}

	// Check if we can process this language
	provider, exists := fp.providers.Get(walkResult.Language)
	if !exists {
		detail.Error = fmt.Sprintf("no provider for language: %s", walkResult.Language)
		return detail
	}

	// Read file content
	content, err := os.ReadFile(walkResult.Path)
	if err != nil {
		detail.Error = fmt.Sprintf("failed to read file: %v", err)
		return detail
	}

	originalContent := string(content)

	// Apply transformation
	result := provider.Transform(originalContent, op.TransformOp)
	if result.Error != nil {
		detail.Error = fmt.Sprintf("transformation failed: %v", result.Error)
		return detail
	}

	detail.MatchCount = result.MatchCount
	detail.Confidence = result.Confidence
	detail.Diff = result.Diff

	// Check if content actually changed
	if result.Modified == originalContent {
		return detail // No changes
	}

	detail.Modified = true
	detail.ModifiedSize = int64(len(result.Modified))

	// Register operation in transaction if safety enabled
	if fp.safetyEnabled && !op.DryRun && tx != nil {
		txOp, err := fp.txManager.AddOperation("modify", walkResult.Path)
		if err != nil {
			detail.Error = fmt.Sprintf("failed to register transaction operation: %v", err)
			return detail
		}
		detail.BackupPath = txOp.BackupPath
	} else if op.Backup {
		// Create backup if requested (when not using transactions)
		backupPath := walkResult.Path + ".bak"
		if err := fp.createBackup(walkResult.Path, backupPath); err != nil {
			detail.Error = fmt.Sprintf("failed to create backup: %v", err)
			return detail
		}
		detail.BackupPath = backupPath
	}

	// Write modified content (unless dry run)
	if !op.DryRun {
		var writeErr error
		if fp.safetyEnabled {
			// Use atomic writer with locking
			writeErr = fp.atomicWriter.WriteFile(walkResult.Path, result.Modified)
		} else {
			// Use simple write
			writeErr = fp.writeFile(walkResult.Path, result.Modified)
		}

		if writeErr != nil {
			detail.Error = fmt.Sprintf("failed to write file: %v", writeErr)

			// Mark operation as failed in transaction
			if fp.safetyEnabled && tx != nil {
				fp.txManager.CompleteOperation(walkResult.Path, writeErr)
			}
			return detail
		}

		// Mark operation as completed in transaction
		if fp.safetyEnabled && tx != nil {
			if err := fp.txManager.CompleteOperation(walkResult.Path, nil); err != nil {
				detail.Error = fmt.Sprintf("failed to complete transaction operation: %v", err)
				return detail
			}
		}
	}

	return detail
}

// createBackup creates a backup copy of the file
func (fp *FileProcessor) createBackup(originalPath, backupPath string) error {
	content, err := os.ReadFile(originalPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, content, 0o644)
}

// writeFile writes content to file with proper permissions
func (fp *FileProcessor) writeFile(path, content string) error {
	// Get original file info for permissions
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	// Write with original permissions
	return os.WriteFile(path, []byte(content), info.Mode())
}

// calculateOverallConfidence computes aggregate confidence across all files
func (fp *FileProcessor) calculateOverallConfidence(details []FileTransformDetail) ConfidenceScore {
	if len(details) == 0 {
		return ConfidenceScore{Score: 0.0, Level: "low"}
	}

	var totalScore float64
	var totalFiles int
	var hasErrors bool
	var hasLowConfidence bool

	for _, detail := range details {
		if detail.Error != "" {
			hasErrors = true
			continue
		}

		if detail.Modified {
			totalScore += detail.Confidence.Score
			totalFiles++

			if detail.Confidence.Score < 0.7 {
				hasLowConfidence = true
			}
		}
	}

	if totalFiles == 0 {
		return ConfidenceScore{Score: 0.0, Level: "low"}
	}

	avgScore := totalScore / float64(totalFiles)

	// Adjust score based on aggregate factors
	factors := []ConfidenceFactor{}

	if hasErrors {
		avgScore -= 0.2
		factors = append(factors, ConfidenceFactor{
			Name:   "file_errors",
			Impact: -0.2,
			Reason: "Some files had processing errors",
		})
	}

	if hasLowConfidence {
		avgScore -= 0.1
		factors = append(factors, ConfidenceFactor{
			Name:   "low_confidence_files",
			Impact: -0.1,
			Reason: "Some files had low confidence transformations",
		})
	}

	if totalFiles > 10 {
		avgScore -= 0.1
		factors = append(factors, ConfidenceFactor{
			Name:   "batch_operation",
			Impact: -0.1,
			Reason: fmt.Sprintf("Large batch operation (%d files)", totalFiles),
		})
	}

	// Clamp score
	if avgScore < 0 {
		avgScore = 0
	} else if avgScore > 1 {
		avgScore = 1
	}

	// Determine level
	level := "high"
	if avgScore < 0.8 {
		level = "medium"
	}
	if avgScore < 0.5 {
		level = "low"
	}

	return ConfidenceScore{
		Score:   avgScore,
		Level:   level,
		Factors: factors,
	}
}

// ValidateChanges verifies that all transformations are valid
func (fp *FileProcessor) ValidateChanges(details []FileTransformDetail) error {
	for _, detail := range details {
		if detail.Error != "" {
			return fmt.Errorf("file %s has error: %s", detail.FilePath, detail.Error)
		}

		if detail.Modified && detail.Confidence.Score < 0.3 {
			return fmt.Errorf("file %s has very low confidence: %.2f",
				detail.FilePath, detail.Confidence.Score)
		}
	}

	return nil
}

// GenerateChecksum creates SHA256 hash of file content for integrity checking
func (fp *FileProcessor) GenerateChecksum(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash), nil
}

// EnableSafety enables/disables safety features at runtime
func (fp *FileProcessor) EnableSafety(enabled bool) {
	fp.safetyEnabled = enabled
}

// IsSafetyEnabled returns current safety status
func (fp *FileProcessor) IsSafetyEnabled() bool {
	return fp.safetyEnabled
}

// Cleanup releases all resources and locks
func (fp *FileProcessor) Cleanup() {
	if fp.atomicWriter != nil {
		fp.atomicWriter.Cleanup()
	}
}
