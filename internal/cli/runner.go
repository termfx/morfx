package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/garaekz/fileman/internal/core"
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/util"
	"github.com/garaekz/fileman/internal/writer"
)

// Runner encapsulates the application's execution logic.
type Runner struct {
	writer writer.Writer
}

// NewRunner creates a new runner with the appropriate writer based on configuration.
func NewRunner(cfg *model.Config) *Runner {
	var w writer.Writer
	if cfg.Operation == model.OpGet {
		// For get operations, use read-only writer that doesn't show any summary
		w = writer.NewReadOnlyWriter()
	} else if cfg.DryRun {
		w = writer.NewDryRunWriter()
	} else if cfg.Interactive {
		w = writer.NewInteractiveWriter()
	} else {
		// Default to staging writer for non-destructive behavior
		w = writer.NewStagingWriter()
	}

	return &Runner{
		writer: w,
	}
}

func (r *Runner) Run(ctx context.Context, files []string, cfg *model.Config) ([]model.Result, error) {
	var (
		results   []model.Result
		resultsMu sync.Mutex
		wg        sync.WaitGroup
		errOnce   sync.Once
		firstErr  error
		stopChan  = make(chan struct{})
	)
	numW := cfg.Workers
	if numW < 1 {
		numW = runtime.NumCPU()
	}
	jobs := make(chan string)

	for i := 0; i < numW; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				case <-ctx.Done():
					return
				case path, ok := <-jobs:
					if !ok {
						return
					}
					res, err := r.processFile(ctx, path, cfg)
					resultsMu.Lock()
					if err != nil {
						errOnce.Do(func() {
							firstErr = err
							close(stopChan) // Circuit breaker: stop all workers
						})
						results = append(results, model.Result{File: path, Success: false, Error: err})
					} else {
						results = append(results, *res)
					}
					resultsMu.Unlock()
				}
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	// Check if any file processing resulted in an error
	if firstErr != nil {
		return results, model.Wrap(model.ErrUnknown, "one or more errors occurred during processing", firstErr)
	}

	// Check if no changes were made and FailIfNoMatch is true
	if len(results) > 0 && cfg.FailIfNoMatch {
		// Count total modified files
		modifiedFilesCount := 0
		for _, res := range results {
			if res.ModifiedCount > 0 {
				modifiedFilesCount++
			}
		}
		if modifiedFilesCount == 0 {
			return results, model.Wrap(model.ErrNoChanges, "no changes made", nil)
		}
	}

	return results, nil
}

// RunHarmless processes a file in a "harmless" mode, simulating changes without writing.
// This is useful for testing or previewing what would happen without modifying the file.
func (r *Runner) RunHarmless(file string, cfg *model.Config) ([]model.Result, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, model.Wrap(model.ErrIO, "reading file", err)
	}
	original := string(data)
	if len(original) == 0 {
		return nil, model.Wrap(model.ErrInvalidInput, "file is empty", nil)
	}

	return []model.Result{
		{
			File:            file,
			Success:         true,
			OriginalContent: original,
			ModifiedContent: original, // No changes in harmless mode
			Changes:         nil,      // No changes made in harmless mode
		},
	}, nil
}

// processFile processes a single file with the provided rules.
func (r *Runner) processFile(ctx context.Context, path string, cfg *model.Config) (*model.Result, error) {
	select {
	case <-ctx.Done():
		return nil, model.Wrap(model.ErrUnknown, "processing cancelled", ctx.Err())
	default:
	}
	var data []byte
	var err error
	// Removed unused stBefore
	var shaBefore string

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		// Validate file accessibility before processing
		f, errOpen := os.OpenFile(path, os.O_RDONLY, 0)
		if errOpen != nil {
			return nil, model.Wrap(model.ErrIO, "file not accessible", errOpen)
		}
		f.Close()
		err = nil // Reset err for next operation
		data, err = os.ReadFile(path)
		if err == nil {
			shaBefore = util.SHA1Hex(data)
		}
	}
	if err != nil {
		return nil, model.Wrap(model.ErrIO, "reading file", err)
	}
	original := string(data)

	manip := core.NewManipulator(cfg)

	// For get operations, use ApplyHarmless to show matches without making changes
	if cfg.Operation == model.OpGet {
		changes, err := manip.ApplyHarmless(original)
		if err != nil {
			if cliErr, ok := err.(model.CLIError); ok {
				return nil, cliErr
			}
			return nil, model.Wrap(model.ErrParseQuery, fmt.Sprintf("applying rule %q", cfg.RuleID), err)
		}

		// Build result for get operation
		res := &model.Result{
			File:            path,
			Time:            time.Now().Format(time.RFC3339),
			SchemaVersion:   model.CurrentSchemaVersion,
			ToolVersion:     model.CurrentToolVersion,
			Success:         true,
			ModifiedCount:   len(changes),
			ChangedBytes:    0, // No actual changes for get operations
			OriginalSHA1:    util.SHA1Hex(data),
			OriginalContent: original,
			ModifiedContent: original, // No modifications for get
			Changes:         changes,
		}
		res.ModifiedSHA1 = res.OriginalSHA1
		return res, nil
	}

	// Early race detection: check content hash before expensive computation
	if path != "-" {
		dataNow, errRead := os.ReadFile(path)
		shaNow := ""
		if errRead == nil {
			shaNow = util.SHA1Hex(dataNow)
		}
		if shaBefore != "" && shaNow != "" && shaBefore != shaNow {
			err = model.Wrap(model.ErrWriteRace, "file content changed before processing", nil)
			return &model.Result{File: path, Success: false, Error: err}, err
		}
	}

	// For non-get operations, use the normal Apply flow
	rewrites, err := manip.Apply(original)
	if err != nil {
		if cliErr, ok := err.(model.CLIError); ok {
			return nil, cliErr
		}
		return nil, model.Wrap(model.ErrParseQuery, fmt.Sprintf("applying rule %q", cfg.RuleID), err)
	}

	modified, changes := core.ApplyRewrites(original, rewrites)

	// Build result
	res := &model.Result{
		File:            path,
		Time:            time.Now().Format(time.RFC3339),
		SchemaVersion:   model.CurrentSchemaVersion,
		ToolVersion:     model.CurrentToolVersion,
		Success:         true,
		ModifiedCount:   len(changes),
		ChangedBytes:    util.SumChangedBytes(changes),
		OriginalSHA1:    util.SHA1Hex(data),
		OriginalContent: original,
		ModifiedContent: modified,
		Changes:         changes,
	}

	// Write back if needed using the Writer interface
	if original != modified && path != "-" {
		// Race detection after processing: check content hash again
		dataAfter, errRead := os.ReadFile(path)
		shaAfter := ""
		if errRead == nil {
			shaAfter = util.SHA1Hex(dataAfter)
		}
		if shaBefore != "" && shaAfter != "" && shaBefore != shaAfter {
			err = model.Wrap(model.ErrWriteRace, "file modified during processing", nil)
			res.Success = false
			res.Error = err
			return res, err
		}

		if err := r.writer.WriteFile(path, []byte(modified), 0o644); err != nil {
			return res, model.Wrap(model.ErrIO, "write file", err)
		}

		// Calculate SHA1 for actual writes (not dry-run)
		if !cfg.DryRun {
			sha1, err := util.SHA1FileHex(path)
			if err != nil {
				err = model.Wrap(model.ErrIO, "calculating SHA1", err)
				res.Success = false
				res.Error = err
				return res, err
			}
			res.ModifiedSHA1 = sha1
		} else {
			res.ModifiedSHA1 = util.SHA1Hex([]byte(modified))
		}
	} else {
		res.ModifiedSHA1 = res.OriginalSHA1
	}

	return res, nil
}

// WriterSummary returns a summary of what the writer did.
func (r *Runner) WriterSummary() string {
	return r.writer.Summary()
}

// ApplyStaged applies staged changes from .morfx/ directory.
func (r *Runner) ApplyStaged() error {
	commitWriter := writer.NewCommitWriter()
	return commitWriter.ApplyStagedChanges()
}

// StagedSummary returns a summary of staged changes that would be applied.
func (r *Runner) StagedSummary() string {
	commitWriter := writer.NewCommitWriter()
	return commitWriter.Summary()
}
