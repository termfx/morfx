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
)

// Runner encapsulates the application's execution logic.
type Runner struct {
	mu sync.RWMutex
}

func (r *Runner) Run(files []string, cfg *model.Config) ([]model.Result, error) {
	totalChanges := 0
	hadError := false

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
					hadError = true
					r.addFileResult(cfg, &results, path, false, nil, err)
					continue
				}
				r.addFileResult(cfg, &results, path, res.Success, res.Changes, nil)
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}

	close(jobs)
	wg.Wait()

	if hadError {
		return nil, model.Wrap(model.ErrUnknown, "errors occurred during processing", nil)
	}
	if totalChanges == 0 && cfg.FailIfNoMatch {
		return nil, model.Wrap(model.ErrNoChanges, "no changes made", nil)
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

	current := original
	var allChanges []model.Change

	manip := core.NewManipulator(cfg)
	modified, changes, err := manip.Apply(current)
	if err != nil {
		if cliErr, ok := err.(model.CLIError); ok {
			return nil, cliErr
		}
		return nil, model.Wrap(model.ErrParseQuery, fmt.Sprintf("applying rule %q", cfg.RuleID), err)
	}

	current = modified
	allChanges = append(allChanges, changes...)

	// Build result
	res := &model.Result{
		File:            path,
		Time:            time.Now().Format(time.RFC3339),
		SchemaVersion:   model.CurrentSchemaVersion,
		ToolVersion:     model.CurrentToolVersion,
		Success:         true,
		ModifiedCount:   len(allChanges),
		ChangedBytes:    util.SumChangedBytes(allChanges),
		OriginalSHA1:    util.SHA1Hex(data),
		OriginalContent: original,
		ModifiedContent: current,
		Changes:         allChanges,
	}

	// Write back if needed
	if original != current && !cfg.DryRun && path != "-" {
		stAfter, _ := os.Stat(path)
		if util.RaceDetected(stBefore, stAfter) {
			err = model.Wrap(model.ErrWriteRace, "file modified during processing", nil)
			res.Success = false
			res.Error = err

			return res, err
		}
		if err := util.WriteFileAtomic(path, []byte(current), 0o644); err != nil {
			return res, model.Wrap(model.ErrIO, "write file", err)
		}
		sha1, err := util.SHA1FileHex(path)
		if err != nil {
			err = model.Wrap(model.ErrIO, "calculating SHA1", err)
			res.Success = false
			res.Error = err
			return res, err
		}
		res.ModifiedSHA1 = sha1
	} else {
		res.ModifiedSHA1 = res.OriginalSHA1
	}

	return res, nil
}

func (r *Runner) addFileResult(cfg *model.Config, results *[]model.Result, path string, succ bool, chgs []model.Change, err error) {
	res := model.Result{
		File:          path,
		Success:       succ,
		Changes:       chgs,
		ModifiedCount: len(chgs),
	}
	if err != nil {
		if ce, ok := err.(model.CLIError); ok {
			res.Error = ce
		} else {
			res.Error = model.Wrap(model.ErrUnknown, "processing file", err)
		}
	}

	*results = append(*results, res)
}
