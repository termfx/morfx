// ===================== internal/cli/runner.go (refactored) =====================
package cli

import (
	"encoding/json"
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
	DryRun      bool
	Verbose     bool
	JSONOutput  bool
	StdoutMode  bool
	ShowDiff    bool
	DiffContext int
	ColorDiff   bool
	Workers     int // for parallel processing, unused in this version
}

// -----------------------------------------------------------------------------
// Public entry points
// -----------------------------------------------------------------------------

// RunWithConfig executes the tool based on a configuration file.
func (r *Runner) RunWithConfig(path string) int {
	cfgBytes, err := os.ReadFile(path)
	if err != nil {
		r.printFatal(core.Wrap(core.ErrIO, "error reading config file", err))
		return 1
	}

	var tc model.ToolConfig
	if err := json.Unmarshal(cfgBytes, &tc); err != nil {
		r.printFatal(core.Wrap(core.ErrParseQuery, "error parsing config file", err))
		return 1
	}

	// Basic validation
	if tc.SchemaVersion != 0 && tc.SchemaVersion > model.CurrentSchemaVersion {
		msg := fmt.Sprintf(
			"config schema version %d is not supported by tool version %s (max schema %d)",
			tc.SchemaVersion, model.CurrentToolVersion, model.CurrentSchemaVersion,
		)
		r.printFatal(core.CLIError{Code: string(model.ECConfigError), Message: msg})
		return 1
	}
	if len(tc.Files) == 0 || len(tc.Rules) == 0 {
		r.printFatal(core.CLIError{Code: string(model.ECConfigError), Message: "config must specify at least one file and one rule."})
		return 2
	}

	files := util.ExpandGlobs(tc.Files)
	return r.run(files, tc.Rules, tc.FailIfNoMatch, tc.ExitCodeNoDiff)
}

// RunWithFlags executes the tool based on CLI flags for a single rule.
func (r *Runner) RunWithFlags(files []string, cfg model.ModificationConfig, failIfNoMatch bool) int {
	return r.run(files, []model.ModificationConfig{cfg}, failIfNoMatch, 2)
}

// -----------------------------------------------------------------------------
// Core execution pipeline
// -----------------------------------------------------------------------------

func (r *Runner) run(files []string, rules []model.ModificationConfig, failIfNoMatch bool, noDiffCode int) int {
	totalChanges := 0
	hadError := false

	var results []model.Result // for JSON output mode

	jobs := make(chan string)

	var wg sync.WaitGroup
	numW := r.Workers
	if numW < 1 {
		numW = runtime.NumCPU()
	}

	for i := 0; i < numW; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				res, err := r.processFileWithRules(path, rules)
				if err != nil {
					r.addFileResult(&results, path, false, nil, err)
					continue
				}
				r.addFileResult(&results, path, res.Success, res.Changes, nil)
			}
		}()
	}

	for _, f := range files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	// Emit output
	if r.JSONOutput {
		b, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(b))
	}

	if hadError {
		return 1
	}
	if totalChanges == 0 && failIfNoMatch {
		return noDiffCode
	}
	return 0
}

// -----------------------------------------------------------------------------
// Single-file processing
// -----------------------------------------------------------------------------

// processFileWithRules contains the corrected logic for handling the 'get' operation.
func (r *Runner) processFileWithRules(path string, rules []model.ModificationConfig) (*model.Result, error) {
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
		return nil, core.Wrap(core.ErrIO, "reading file", err)
	}
	original := string(data)

	// --- Handle 'get' operation as a special read-only case ---
	if len(rules) == 1 && rules[0].Operation == model.OpGet {
		manip := core.NewManipulator(rules[0])
		_, changes, err := manip.Apply(original)
		if err != nil {
			return nil, err
		}

		if len(changes) > 0 {
			// For 'get', we print the content of the first matched node.
			fmt.Print(changes[0].Original)
		}
		// The 'get' operation terminates here successfully without writing files.
		return &model.Result{
			File:          path,
			Success:       true,
			ModifiedCount: len(changes),
		}, nil
	}

	// --- Regular modification flow for replace, delete, etc. ---
	current := original
	var allChanges []model.Change
	for _, rule := range rules {
		manip := core.NewManipulator(rule)
		modified, changes, err := manip.Apply(current)
		if err != nil {
			if cliErr, ok := err.(core.CLIError); ok {
				return nil, cliErr
			}
			return nil, core.Wrap(core.ErrParseQuery, fmt.Sprintf("applying rule %q", rule.RuleID), err)
		}
		if err := validateRuleContracts(rule, changes); err != nil {
			return nil, core.Wrap(core.ErrParseQuery, fmt.Sprintf("contract for rule %q failed", rule.RuleID), err)
		}
		current = modified
		allChanges = append(allChanges, changes...)
	}

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
	if original != current && !r.DryRun && path != "-" {
		stAfter, _ := os.Stat(path)
		if util.RaceDetected(stBefore, stAfter) {
			res.Success = false
			res.ErrorCode = model.ECWriteRace
			res.Error = model.ErrWriteRace.Error()
			return res, model.ErrWriteRace
		}
		if err := util.WriteFileAtomic(path, []byte(current), 0o644); err != nil {
			res.Success = false
			res.ErrorCode = model.ECWriteError
			res.Error = err.Error()
			return res, core.Wrap(core.ErrIO, "write file", err)
		}
		sha1, err := util.SHA1FileHex(path)
		if err != nil {
			res.Success = false
			res.ErrorCode = model.ECWriteError
			res.Error = err.Error()
			return res, core.Wrap(core.ErrIO, "write file", err)
		}
		res.ModifiedSHA1 = sha1
	} else {
		res.ModifiedSHA1 = res.OriginalSHA1
	}

	return res, nil
}

// -----------------------------------------------------------------------------
// Output helpers
// -----------------------------------------------------------------------------

func (r *Runner) addFileResult(results *[]model.Result, path string, succ bool, chgs []model.Change, err error) {
	res := model.Result{
		File:          path,
		Success:       succ,
		Changes:       chgs,
		ModifiedCount: len(chgs),
	}
	if err != nil {
		if ce, ok := err.(core.CLIError); ok {
			res.ErrorCode = model.ErrorCode(ce.Code)
			res.Error = ce.Message
		} else {
			res.ErrorCode = model.ECUnknown
			res.Error = err.Error()
		}
	}

	if !r.JSONOutput {
		r.printResultCLI(&res)
	}
	*results = append(*results, res)
}

func (r *Runner) printResultCLI(res *model.Result) {
	if !res.Success {
		fmt.Fprintf(os.Stderr, "✗ %s: %s (%s)\n", res.File, res.Error, res.ErrorCode)
		return
	}

	if r.Verbose {
		if res.ModifiedCount > 0 {
			fmt.Printf("✓ %s — %d changes (%d bytes diff)\n", res.File, res.ModifiedCount, res.ChangedBytes)
		} else {
			fmt.Printf("✓ %s — No changes\n", res.File)
		}
	}

	if r.ShowDiff && res.ModifiedCount > 0 {
		diff := util.UnifiedDiff(res.OriginalContent, res.ModifiedContent, res.File, r.DiffContext, r.ColorDiff)
		fmt.Print(diff)
	}

	if r.StdoutMode {
		fmt.Print(res.ModifiedContent)
	}
}

func (r *Runner) printFatal(err error) {
	if r.JSONOutput {
		if ce, ok := err.(core.CLIError); ok {
			fmt.Println(ce.JSON())
		} else {
			b, _ := json.Marshal(core.CLIError{Code: string(model.ECUnknown), Message: err.Error()})
			fmt.Println(string(b))
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

// -----------------------------------------------------------------------------
// Contracts
// -----------------------------------------------------------------------------

func validateRuleContracts(rule model.ModificationConfig, changes []model.Change) error {
	if rule.MustMatch > 0 && len(changes) < rule.MustMatch {
		return model.ErrMustMatchFailed
	}
	if rule.MustChangeBytes > 0 && util.SumChangedBytes(changes) < rule.MustChangeBytes {
		return model.ErrMustChangeBytesFailed
	}
	return nil
}
