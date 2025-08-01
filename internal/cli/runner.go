package cli

import (
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
	mu     sync.RWMutex
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

func (r *Runner) Run(files []string, cfg *model.Config) ([]model.Result, error) {
	var results []model.Result // for JSON output mode

	jobs := make(chan string)

	var wg sync.WaitGroup
	numW := cfg.Workers
	if numW < 1 {
		numW = runtime.NumCPU()
	}

	for i := 0; i < numW; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				res, err := r.processFile(path, cfg)
				if err != nil {
					// Log the error but continue processing other files
					fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", path, err)
					results = append(results, model.Result{File: path, Success: false, Error: err})
					continue
				}
				results = append(results, *res)
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}

	close(jobs)
	wg.Wait()

	// Check if any file processing resulted in an error
	for _, res := range results {
		if !res.Success {
			return results, model.Wrap(model.ErrUnknown, "one or more errors occurred during processing", nil)
		}
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
func (r *Runner) processFile(path string, cfg *model.Config) (*model.Result, error) {
	var data []byte
	var err error
	var stBefore os.FileInfo

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		stBefore, err = os.Stat(path)
		if err == nil {
			data, err = os.ReadFile(path)
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
		stAfter, _ := os.Stat(path)
		if util.RaceDetected(stBefore, stAfter) {
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
