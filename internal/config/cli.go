package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/garaekz/fileman/internal/model"
)

type SecurityThresholds struct {
	Low    int
	Medium int
	High   int
}

var secThresholds = SecurityThresholds{
	Low:    10,
	Medium: 50,
	High:   100,
}

// BuildConfigFromFlags parses command-line flags and builds a configuration
func BuildConfigFromFlags(args []string) (*model.Config, []string, error) {
	fs := pflag.NewFlagSet("morfx", pflag.ContinueOnError)
	fs.Usage = func() {
		PrintUsage(fs)
	}

	// Define flags
	fs.BoolP("help", "h", true, "Show this help message and exit.")
	fs.StringP(
		"query",
		"q",
		"",
		"DSL query for node selection (e.g., 'func:MyFunc > call:os.Getenv'). (Required)",
	)
	operation := fs.StringP(
		"op",
		"o",
		"get",
		"Operation: get, replace, delete, insert-before, insert-after.",
	)
	fs.StringP("repl", "r", "", "Replacement string for replace/insert operations.")

	fs.StringP(
		"lang",
		"l",
		"",
		"Target language (go, python, etc.). Inferred from file extensions if omitted.",
	)
	fs.BoolP(
		"include-tests",
		"t",
		false,
		"Include test files (*_test.go, etc.) in the operation.",
	)
	workers := fs.IntP(
		"workers",
		"w",
		0,
		"Number of concurrent workers, 0 means use all available CPUs. (Default: 0).",
	)

	fs.BoolP(
		"force",
		"f",
		false,
		"Force apply changes without confirmation for medium-sized refactors.",
	)
	fs.BoolP("dry-run", "d", false, "Perform a trial run without writing any files.")
	fs.Bool(
		"commit",
		false,
		"Write changes to disk (overrides default dry-run behavior).",
	)
	showDiff := fs.BoolP("diff", "D", false, "Show a unified diff of the changes.")
	diffContext := fs.IntP("diff-context", "C", 3, "Lines of context for the diff.")
	verbose := fs.BoolP("verbose", "v", false, "Enable verbose output.")
	jsonOutput := fs.BoolP("json", "j", false, "Output results in JSON format.")
	stdout := fs.Bool(
		"stdout",
		false,
		"Output modified content to stdout instead of writing to files.",
	)

	// New flags for recursive scanning and filtering
	fs.String("root", "", "Root directory for scanning (default: current directory).")
	fs.StringSlice("include", nil, "Include file patterns (glob).")
	fs.StringSlice("exclude", nil, "Exclude file patterns (glob).")
	fs.Bool("no-gitignore", false, "Disable .gitignore filtering.")
	fs.Int64("max-bytes", 5*1024*1024, "Maximum file size to process (default: 5MB).")
	fs.Bool(
		"follow-symlinks",
		false,
		"Follow symbolic links during directory traversal.",
	)
	fs.Bool("stdin", false, "Read replacement content from stdin.")

	// Parse the flags
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	op := model.Operation(*operation)
	cfg := &model.Config{
		RuleID:      fmt.Sprintf("cli-operation-%s", op),
		Operation:   op,
		ShowDiff:    *showDiff,
		DiffContext: *diffContext,
		Verbose:     *verbose,
		JSONOutput:  *jsonOutput,
		StdoutMode:  *stdout,
		Workers:     *workers,
	}

	return validateFlags(fs, cfg)
}

func validateFlags(fs *pflag.FlagSet, cfg *model.Config) (*model.Config, []string, error) {
	// If the --help flag is set, show usage and return an error
	if !fs.HasFlags() || fs.Changed("help") {
		fs.Usage()
		return nil, nil, flag.ErrHelp
	}

	// Check commit flag first
	if cfg, ok := checkCommit(fs); ok {
		return cfg, nil, nil
	}

	// Check query flag
	query, ok := checkQuery(fs)
	if !ok {
		return nil, nil, fmt.Errorf("query flag is required")
	}

	// Get targets
	targets, err := resolveTargets(fs)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving targets: %w", err)
	}

	replacement, err := resolveReplacement(fs)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving replacement text: %w", err)
	}

	provider, files, err := resolveProviderAndFiles(fs, targets)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving language provider: %w", err)
	}

	// Determine the execution mode
	isDryRun := false // Default to staging mode (non-destructive)
	isInteractive := false

	if fs.Changed("dry-run") {
		isDryRun = true       // Explicit --dry-run always enables dry-run
		isInteractive = false // and disables interactive mode
	}

	// For 'get' operations, don't fail if no matches found
	failIfNoMatch := !fs.Changed("force") && cfg.Operation != model.OpGet

	// Set the rest of the configuration
	cfg.Pattern = query
	cfg.Replacement = replacement
	cfg.Provider = provider
	cfg.DryRun = isDryRun
	cfg.Interactive = isInteractive
	cfg.FailIfNoMatch = failIfNoMatch

	// Apply test file filtering if needed
	if !fs.Changed("include-tests") {
		ignorePatterns, _ := provider.GetDefaultIgnorePatterns()
		files = filterFiles(files, ignorePatterns)
	}

	diffThreshold := secThresholds.Medium
	if len(files) > diffThreshold && !isDryRun {
		fmt.Fprintf(
			os.Stdout,
			"Warning: This operation affects %d files. Forcing diff preview.\n",
			len(files),
		)
		cfg.ShowDiff = true
	}

	return cfg, files, nil
}

func filterFiles(files []string, ignorePatterns []string) []string {
	var filtered []string
	for _, file := range files {
		isIgnored := false
		for _, pattern := range ignorePatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
				isIgnored = true
				break
			}
		}
		if !isIgnored {
			filtered = append(filtered, file)
		}
	}
	return filtered
}
