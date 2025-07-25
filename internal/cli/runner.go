package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
}

// RunWithConfig executes the tool based on a configuration file.
func (r *Runner) RunWithConfig(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		return 1
	}

	var tc model.ToolConfig
	if err := json.Unmarshal(b, &tc); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config file: %v\n", err)
		return 1
	}

	// Validate config
	if tc.SchemaVersion != 0 && tc.SchemaVersion > model.CurrentSchemaVersion {
		fmt.Fprintf(os.Stderr, "Config schema version %d is not supported by tool version %s (max schema %d)\n",
			tc.SchemaVersion, model.CurrentToolVersion, model.CurrentSchemaVersion)
		return 1
	}
	if len(tc.Files) == 0 || len(tc.Rules) == 0 {
		fmt.Fprintln(os.Stderr, "Config must specify at least one file and one rule.")
		return 2
	}

	expandedFiles := util.ExpandGlobs(tc.Files)
	totalChanges := 0
	hadError := false

	for _, file := range expandedFiles {
		res, err := r.processFileWithRules(file, tc.Rules)
		if err != nil {
			hadError = true
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", file, err)
			continue
		}

		r.printResult(res)
		if !res.Success {
			hadError = true
		} else {
			totalChanges += res.ModifiedCount
		}
	}

	if hadError {
		return 1
	}
	if totalChanges == 0 && tc.FailIfNoMatch {
		if tc.ExitCodeNoDiff != 0 {
			return tc.ExitCodeNoDiff
		}
		return 2
	}
	return 0
}

// RunWithFlags executes the tool based on command-line flags for a single rule.
func (r *Runner) RunWithFlags(files []string, cfg model.ModificationConfig, failIfNoMatch bool) int {
	totalChanges := 0
	hadError := false

	for _, file := range files {
		res, err := r.processFileWithRules(file, []model.ModificationConfig{cfg})
		if err != nil {
			hadError = true
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", file, err)
			continue
		}

		r.printResult(res)
		if !res.Success {
			hadError = true
		} else {
			totalChanges += res.ModifiedCount
		}
	}

	if hadError {
		return 1
	}
	if totalChanges == 0 && failIfNoMatch {
		return 2
	}
	return 0
}

// processFileWithRules reads a single file and applies a pipeline of rules.
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
		return nil, fmt.Errorf("reading file: %w", err)
	}

	originalContent := string(data)
	currentContent := originalContent
	var allChanges []model.Change

	for _, rule := range rules {
		manipulator := core.NewManipulator(rule)
		modified, changes, err := manipulator.Apply(currentContent)
		if err != nil {
			return nil, fmt.Errorf("applying rule %q: %w", rule.RuleID, err)
		}

		// Per-rule contract validation
		if err := validateRuleContracts(rule, changes); err != nil {
			return nil, fmt.Errorf("contract for rule %q failed: %w", rule.RuleID, err)
		}

		currentContent = modified
		allChanges = append(allChanges, changes...)
	}

	// Create result object
	res := &model.Result{
		File:            path,
		Time:            time.Now().Format(time.RFC3339),
		SchemaVersion:   model.CurrentSchemaVersion,
		ToolVersion:     model.CurrentToolVersion,
		Success:         true,
		ModifiedCount:   len(allChanges),
		ChangedBytes:    util.SumChangedBytes(allChanges),
		OriginalSHA1:    util.SHA1Hex(data),
		OriginalContent: originalContent,
		ModifiedContent: currentContent,
	}

	// Write file if modified and not in dry-run mode
	if originalContent != currentContent && !r.DryRun && path != "-" {
		// Race condition check
		stAfter, _ := os.Stat(path)
		if util.RaceDetected(stBefore, stAfter) {
			res.Success = false
			res.Error = model.ErrWriteRace.Error()
			res.ErrorCode = model.ECWriteRace
			return res, model.ErrWriteRace
		}

		if err := util.WriteFileAtomic(path, []byte(currentContent), 0o644); err != nil {
			res.Success = false
			res.Error = fmt.Sprintf("write error: %v", err)
			res.ErrorCode = model.ECWriteError
			return res, err
		}
		res.ModifiedSHA1 = util.SHA1FileHex(path)
	} else if originalContent == currentContent {
		res.ModifiedSHA1 = res.OriginalSHA1
	}

	return res, nil
}

// printResult formats and prints the result to the console.
func (r *Runner) printResult(res *model.Result) {
	if r.JSONOutput {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		return
	}

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

// validateRuleContracts checks if a rule's Must* conditions are met.
func validateRuleContracts(rule model.ModificationConfig, changes []model.Change) error {
	if rule.MustMatch > 0 && len(changes) < rule.MustMatch {
		return model.ErrMustMatchFailed
	}
	if rule.MustChangeBytes > 0 && util.SumChangedBytes(changes) < rule.MustChangeBytes {
		return model.ErrMustChangeBytesFailed
	}
	return nil
}
