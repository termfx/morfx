package cli

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

	"github.com/termfx/morfx/internal/db"
	"github.com/termfx/morfx/internal/manipulator"
	"github.com/termfx/morfx/internal/model"
	"github.com/termfx/morfx/internal/writer"
)

type Output struct {
	Results        []model.Result
	ExitCode       int
	FileErrorCount int
	Error          error
}

// Run executes the file manipulation based on the provided configuration and files.
func Run(files []string, cfg *model.Config) Output {
	if cfg.Operation == model.OpCommit {
		return commit()
	}

	return process(files, cfg)
}

func commit() Output {
	commitWriter := writer.NewCommitWriter()
	err := commitWriter.ApplyStagedChanges()
	if err != nil {
		return Output{
			ExitCode: 1,
			Error:    fmt.Errorf("error applying staged changes: %w", err),
		}
	}
	// After applying staged changes, checkpoint WAL if needed
	if dbConn, dbErr := db.Open(); dbErr == nil {
		_ = db.CheckWALSizeAndCheckpoint(dbConn.DB)
		dbConn.Close()
	}

	return Output{}
}

func process(files []string, cfg *model.Config) Output {
	var results []model.Result // for JSON output mode
	var errorCount int
	jobs := make(chan string)

	var wg sync.WaitGroup
	var mu sync.Mutex // Protects results and errorCount
	numW := cfg.Workers
	if numW < 1 {
		numW = runtime.NumCPU()
	}

	for i := 0; i < numW; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				res, err := processFile(path, cfg)
				mu.Lock()
				if err != nil {
					// Log the error but continue processing other files
					fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", path, err)
					results = append(results, model.Result{File: path, Success: false, Error: err})
					errorCount++
				} else {
					results = append(results, *res)
				}
				mu.Unlock()
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}

	close(jobs)
	wg.Wait()

	if errorCount > 0 {
		return Output{
			Results:        results,
			ExitCode:       2,
			FileErrorCount: errorCount,
			Error:          fmt.Errorf("encountered %d errors during processing", errorCount),
		}
	}

	return Output{
		Results: results,
	}
}

// processFile processes a single file with the provided rules.
func processFile(path string, cfg *model.Config) (*model.Result, error) {
	var data []byte
	var err error

	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, model.Wrap(model.ErrIO, "reading file", err)
	}
	original := string(data)

	return manipulator.Manipulate(cfg, path, original, data)
}
