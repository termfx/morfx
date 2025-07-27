package cli

import (
	"os"
	"sync"

	"github.com/garaekz/fileman/internal/model"
)

// Runner encapsulates the application's execution logic.
type Runner struct {
	mu sync.RWMutex
}

// func (r *Runner) Run(files []string, cfg *model.Config) []model.Result {
// 	// totalChanges := 0
// 	// hadError := false

// 	var results []model.Result // for JSON output mode

// 	jobs := make(chan string)

// 	var wg sync.WaitGroup
// 	numW := cfg.Workers
// 	if numW < 1 {
// 		numW = runtime.NumCPU()
// 	}

// 	for i := 0; i < numW; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			// for path := range jobs {
// 			// 	_, err := r.processFile(path, cfg)
// 			// 	if err != nil {
// 			// 		return // early if any error occurs
// 			// 	}
// 			// 	// if err != nil {
// 			// 	// 	r.addFileResult(&results, path, false, nil, err)
// 			// 	// 	continue
// 			// 	// }
// 			// 	// r.addFileResult(&results, path, res.Success, res.Changes, nil)
// 			// }
// 		}()
// 	}

// 	for _, f := range files {
// 		jobs <- f
// 	}
// 	close(jobs)
// 	wg.Wait()

// 	return results
// }

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
// func (r *Runner) processFile(path string, cfg *model.Config) (*model.Result, error) {
// 	var data []byte
// 	var err error
// 	// var stBefore os.FileInfo

// 	if path == "-" {
// 		data, err = io.ReadAll(os.Stdin)
// 	} else {
// 		// stBefore, err = os.Stat(path)
// 		if err == nil {
// 			data, err = os.ReadFile(path)
// 		}
// 	}
// 	if err != nil {
// 		return nil, model.Wrap(model.ErrIO, "reading file", err)
// 	}

// 	original := string(data)
// 	if len(original) == 0 {
// 		return nil, model.Wrap(model.ErrInvalidInput, "file is empty", nil)
// 	}

// 	return nil, nil // TODO: Implement the actual file processing logic here
// }

// // --- Handle 'get' operation as a special read-only case ---
// if len(rules) == 1 && rules[0].Operation == model.OpGet {
// 	manip := core.NewManipulator(rules[0])
// 	_, changes, err := manip.Apply(original)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(changes) > 0 {
// 		// For 'get', we print the content of the first matched node.
// 		fmt.Print(changes[0].Original)
// 	}
// 	// The 'get' operation terminates here successfully without writing files.
// 	return &model.Result{
// 		File:          path,
// 		Success:       true,
// 		ModifiedCount: len(changes),
// 	}, nil
// }

// // --- Regular modification flow for replace, delete, etc. ---
// current := original
// var allChanges []model.Change
// for _, rule := range rules {
// 	manip := core.NewManipulator(rule)
// 	modified, changes, err := manip.Apply(current)
// 	if err != nil {
// 		if cliErr, ok := err.(core.CLIError); ok {
// 			return nil, cliErr
// 		}
// 		return nil, core.Wrap(core.ErrParseQuery, fmt.Sprintf("applying rule %q", rule.RuleID), err)
// 	}
// 	if err := validateRuleContracts(rule, changes); err != nil {
// 		return nil, core.Wrap(core.ErrParseQuery, fmt.Sprintf("contract for rule %q failed", rule.RuleID), err)
// 	}
// 	current = modified
// 	allChanges = append(allChanges, changes...)
// }

// // Build result
// res := &model.Result{
// 	File:            path,
// 	Time:            time.Now().Format(time.RFC3339),
// 	SchemaVersion:   model.CurrentSchemaVersion,
// 	ToolVersion:     model.CurrentToolVersion,
// 	Success:         true,
// 	ModifiedCount:   len(allChanges),
// 	ChangedBytes:    util.SumChangedBytes(allChanges),
// 	OriginalSHA1:    util.SHA1Hex(data),
// 	OriginalContent: original,
// 	ModifiedContent: current,
// 	Changes:         allChanges,
// }

// // Write back if needed
// if original != current && !r.DryRun && path != "-" {
// 	stAfter, _ := os.Stat(path)
// 	if util.RaceDetected(stBefore, stAfter) {
// 		res.Success = false
// 		res.ErrorCode = model.ECWriteRace
// 		res.Error = model.ErrWriteRace.Error()
// 		return res, model.ErrWriteRace
// 	}
// 	if err := util.WriteFileAtomic(path, []byte(current), 0o644); err != nil {
// 		res.Success = false
// 		res.ErrorCode = model.ECWriteError
// 		res.Error = err.Error()
// 		return res, core.Wrap(core.ErrIO, "write file", err)
// 	}
// 	sha1, err := util.SHA1FileHex(path)
// 	if err != nil {
// 		res.Success = false
// 		res.ErrorCode = model.ECWriteError
// 		res.Error = err.Error()
// 		return res, core.Wrap(core.ErrIO, "write file", err)
// 	}
// 	res.ModifiedSHA1 = sha1
// } else {
// 	res.ModifiedSHA1 = res.OriginalSHA1
// }

// return res, nil
// }

// // -----------------------------------------------------------------------------
// // Output helpers
// // -----------------------------------------------------------------------------
// func (r *Runner) addFileResult(results *[]model.Result, path string, succ bool, chgs []model.Change, err error) {
// 	res := model.Result{
// 		File:          path,
// 		Success:       succ,
// 		Changes:       chgs,
// 		ModifiedCount: len(chgs),
// 	}
// 	if err != nil {
// 		if ce, ok := err.(core.CLIError); ok {
// 			res.ErrorCode = model.ErrorCode(ce.Code)
// 			res.Error = ce.Message
// 		} else {
// 			res.ErrorCode = model.ECUnknown
// 			res.Error = err.Error()
// 		}
// 	}

// 	if !r.JSONOutput {
// 		r.printResultCLI(&res)
// 	}
// 	*results = append(*results, res)
// }
// // -----------------------------------------------------------------------------
// // Contracts
// // -----------------------------------------------------------------------------

// func validateRuleContracts(rule model.ModificationConfig, changes []model.Change) error {
// 	if rule.MustMatch > 0 && len(changes) < rule.MustMatch {
// 		return model.ErrMustMatchFailed
// 	}
// 	if rule.MustChangeBytes > 0 && util.SumChangedBytes(changes) < rule.MustChangeBytes {
// 		return model.ErrMustChangeBytesFailed
// 	}
// 	return nil
// }
