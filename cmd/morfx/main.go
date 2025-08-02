package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/garaekz/fileman/internal/cli"
	"github.com/garaekz/fileman/internal/lang"
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/provider"
	"github.com/garaekz/fileman/internal/scanner"
	"github.com/garaekz/fileman/internal/util"
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

// main is the entry point for morfx, the command-line tool for file transformations.
// It parses command-line flags, builds a configuration, and runs the transformation.
func main() {
	cfg, files, err := buildConfigFromFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	runner := cli.NewRunner(cfg)
	var res []model.Result
	ctx := context.Background()

	// Handle special cases
	if cfg.RuleID == "cli-commit" {
		// Apply staged changes
		err = runner.ApplyStaged()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying staged changes: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(runner.StagedSummary())
		return
	} else if cfg.Operation == model.OpGet && cfg.RuleID == "cli-first-arg" {
		res, err = runner.RunHarmless(files[0], cfg)
	} else {
		res, err = runner.Run(ctx, files, cfg)
	}
	handleOutputAndExit(res, err, cfg, runner)
}

// buildConfigFromFlags parses command-line flags and builds a configuration
func buildConfigFromFlags(args []string) (*model.Config, []string, error) {
	fs := pflag.NewFlagSet("morfx", pflag.ContinueOnError)
	fs.Usage = func() {
		printUsage(fs)
	}

	// Define flags
	fs.BoolP("help", "h", true, "Show this help message and exit.")
	query := fs.StringP("query", "q", "", "DSL query for node selection (e.g., 'func:MyFunc > call:os.Getenv'). (Required)")
	operation := fs.StringP("op", "o", "get", "Operation: get, replace, delete, insert-before, insert-after.")
	replacement := fs.StringP("repl", "r", "", "Replacement string for replace/insert operations.")

	langFlag := fs.StringP("lang", "l", "", "Target language (go, python, etc.). Inferred from file extensions if omitted.")
	includeTests := fs.BoolP("include-tests", "t", false, "Include test files (*_test.go, etc.) in the operation.")
	workers := fs.IntP("workers", "w", 0, "Number of concurrent workers, 0 means use all available CPUs. (Default: 0).")

	force := fs.BoolP("force", "f", false, "Force apply changes without confirmation for medium-sized refactors.")
	dryRun := fs.BoolP("dry-run", "d", false, "Perform a trial run without writing any files.")
	commit := fs.Bool("commit", false, "Write changes to disk (overrides default dry-run behavior).")
	showDiff := fs.BoolP("diff", "D", false, "Show a unified diff of the changes.")
	diffContext := fs.IntP("diff-context", "C", 3, "Lines of context for the diff.")
	verbose := fs.BoolP("verbose", "v", false, "Enable verbose output.")
	jsonOutput := fs.BoolP("json", "j", false, "Output results in JSON format.")
	stdout := fs.Bool("stdout", false, "Output modified content to stdout instead of writing to files.")

	// New flags for recursive scanning and filtering
	root := fs.String("root", "", "Root directory for scanning (default: current directory).")
	includeGlobs := fs.StringSlice("include", nil, "Include file patterns (glob).")
	excludeGlobs := fs.StringSlice("exclude", nil, "Exclude file patterns (glob).")
	noGitignore := fs.Bool("no-gitignore", false, "Disable .gitignore filtering.")
	maxBytes := fs.Int64("max-bytes", 5*1024*1024, "Maximum file size to process (default: 5MB).")
	followSymlinks := fs.Bool("follow-symlinks", false, "Follow symbolic links during directory traversal.")
	stdin := fs.Bool("stdin", false, "Read replacement content from stdin.")

	// Parse the flags
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	// If the --help flag is set, show usage and return an error
	if !fs.HasFlags() || fs.Changed("help") {
		fs.Usage()
		return nil, nil, flag.ErrHelp
	}

	// If no query is provided, check for special modes
	if *query == "" {
		// Check if this is a commit operation
		if *commit {
			// Special config for commit mode - no files needed
			commitCfg := &model.Config{
				RuleID:      "cli-commit",
				Operation:   model.OpGet, // Dummy operation
				DryRun:      false,
				Interactive: false,
			}
			return commitCfg, []string{}, nil
		}

		// Only allow single file mode when no query is provided
		if fs.NArg() == 1 {
			stat, err := os.Stat(fs.Arg(0))
			if os.IsNotExist(err) {
				return nil, nil, errors.New("the specified file does not exist")
			}

			// Only use the shortcut for regular files, not directories
			if stat.Mode().IsRegular() && filepath.Ext(fs.Arg(0)) != "" {
				// Try to resolve the language provider based on the file extension
				provider, err := lang.ResolveProvider("", fs.Args())
				if err != nil {
					return nil, nil, err
				}
				// We have to clean the file and see if thats not a filtered file.
				ignorePatterns, _ := provider.GetDefaultIgnorePatterns()
				files := filterFiles(fs.Args(), ignorePatterns)
				if len(files) == 0 {
					return nil, nil, fmt.Errorf("the only provided file %s is ignored by the default ignore patterns", fs.Arg(0))
				}

				specialCfg := &model.Config{
					RuleID:    "cli-first-arg", // Special rule ID for the first argument case
					Provider:  provider,
					Operation: model.Operation(*operation), // Default to 'get' operation
				}

				return specialCfg, files, nil
			}
		}
		return nil, nil, errors.New("the --query flag is required")
	}

	// Handle targets (files or directories)
	targets := fs.Args()
	if len(targets) == 0 && *root == "" {
		// Default to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, nil, fmt.Errorf("getting current directory: %w", err)
		}
		targets = []string{cwd}
	} else if *root != "" {
		targets = []string{*root}
	}

	// Handle stdin input for replacement
	replacementText := *replacement
	if *stdin {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			stdinBytes, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read from stdin: %w", err)
			}
			if len(stdinBytes) > 0 {
				replacementText = string(stdinBytes)
			}
		}
	}

	// First, scan for files to determine the language if not specified
	var provider provider.LanguageProvider
	var files []string
	var err error

	if *langFlag != "" {
		// Language explicitly specified, resolve it directly
		provider, err = lang.ResolveProvider(*langFlag, []string{})
		if err != nil {
			return nil, nil, err
		}

		// Use scanner to discover files
		scannerCfg := scanner.Config{
			MaxBytes:       *maxBytes,
			FollowSymlinks: *followSymlinks,
			IncludeGlobs:   *includeGlobs,
			ExcludeGlobs:   *excludeGlobs,
			NoGitignore:    *noGitignore,
			Provider:       provider,
		}

		s := scanner.New(scannerCfg)
		files, err = s.ScanTargets(context.Background(), targets)
		if err != nil {
			return nil, nil, fmt.Errorf("scanning targets: %w", err)
		}
	} else {
		// Language not specified, scan first to find files, then infer language
		scannerCfg := scanner.Config{
			MaxBytes:       *maxBytes,
			FollowSymlinks: *followSymlinks,
			IncludeGlobs:   *includeGlobs,
			ExcludeGlobs:   *excludeGlobs,
			NoGitignore:    *noGitignore,
			Provider:       nil, // No provider yet
		}

		s := scanner.New(scannerCfg)
		files, err = s.ScanTargets(context.Background(), targets)
		if err != nil {
			return nil, nil, fmt.Errorf("scanning targets: %w", err)
		}

		if len(files) == 0 {
			return nil, nil, errors.New("no files found matching the criteria")
		}

		// Now resolve provider from discovered files
		provider, err = lang.ResolveProvider("", files)
		if err != nil {
			return nil, nil, err
		}
	}

	if len(files) == 0 {
		return nil, nil, errors.New("no files found matching the criteria")
	}

	// Translate query
	tsQuery, err := translateQuery(*query, provider)
	if err != nil {
		return nil, nil, err
	}

	// Determine the execution mode
	isDryRun := false // Default to staging mode (non-destructive)
	isInteractive := false

	// Note: --commit with query should still use staging mode
	// Only --commit without query applies staged changes

	if *dryRun {
		isDryRun = true       // Explicit --dry-run always enables dry-run
		isInteractive = false // and disables interactive mode
	}

	// For 'get' operations, don't fail if no matches found
	failIfNoMatch := !*force && model.Operation(*operation) != model.OpGet

	cfg := &model.Config{
		RuleID:        "cli-operation",
		Pattern:       tsQuery,
		Replacement:   replacementText,
		Operation:     model.Operation(*operation),
		Provider:      provider,
		DryRun:        isDryRun,
		Interactive:   isInteractive,
		ShowDiff:      *showDiff,
		DiffContext:   *diffContext,
		Verbose:       *verbose,
		JSONOutput:    *jsonOutput,
		StdoutMode:    *stdout,
		FailIfNoMatch: failIfNoMatch,
		Workers:       *workers,
	}

	// Apply test file filtering if needed
	if !*includeTests {
		ignorePatterns, _ := provider.GetDefaultIgnorePatterns()
		files = filterFiles(files, ignorePatterns)
	}

	diffThreshold := secThresholds.Medium
	if len(files) > diffThreshold && !isDryRun {
		fmt.Fprintf(os.Stdout, "Warning: This operation affects %d files. Forcing diff preview.\n", len(files))
		cfg.ShowDiff = true
	}

	return cfg, files, nil
}

// translateQuery translates the DSL query into a format understood by the provider.
func translateQuery(query string, provider provider.LanguageProvider) (string, error) {
	// Use the provider's DSL translator to convert the query
	tsQuery, err := provider.TranslateDSL(query)
	if err != nil {
		return "", fmt.Errorf("failed to translate query: %w", err)
	}

	return tsQuery, nil
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

func handleOutputAndExit(res []model.Result, err error, cfg *model.Config, runner *cli.Runner) {
	if err != nil {
		printFatal(err, cfg.JSONOutput)
	}
	for _, r := range res {
		printResultCLI(&r, cfg)
	}

	// Print writer summary if not in JSON mode and not stdout mode
	if !cfg.JSONOutput && !cfg.StdoutMode {
		summary := runner.WriterSummary()
		if summary != "" {
			fmt.Fprintf(os.Stderr, "\n%s", summary)
		}
	}
}

func printResultCLI(res *model.Result, cfg *model.Config) {
	if !res.Success {
		cliErr, ok := res.Error.(model.CLIError)
		if !ok {
			cliErr = model.CLIError{
				Code:    model.ErrUnknown,
				Message: res.Error.Error(),
			}
		}
		fmt.Fprintf(os.Stderr, "✗ %s: %s (%s)\n", res.File, cliErr, cliErr.Message)
		return
	}

	if cfg.RuleID == "cli-first-arg" {
		// Special case for the first argument rule
		fmt.Printf("✓ %s — No changes made (harmless mode)\n", res.File)
		diff := util.UnifiedDiff("", res.ModifiedContent, res.File, cfg.DiffContext)
		fmt.Print(diff)
		return
	}

	if cfg.Verbose {
		if res.ModifiedCount > 0 {
			fmt.Printf("✓ %s — %d changes (%d bytes diff)\n", res.File, res.ModifiedCount, res.ChangedBytes)
			// For get operations, show the matched content
			if cfg.Operation == model.OpGet {
				for i, change := range res.Changes {
					fmt.Printf("  Match %d: '%s' (lines %d-%d)\n", i+1, change.New, change.LineStart, change.LineEnd)
				}
			}
		} else {
			fmt.Printf("✓ %s — No changes\n", res.File)
		}
		return
	}

	if cfg.ShowDiff && res.ModifiedCount > 0 {
		diff := util.UnifiedDiff(res.OriginalContent, res.ModifiedContent, res.File, cfg.DiffContext)
		fmt.Print(diff)
		return
	}

	if cfg.StdoutMode {
		fmt.Print(res.ModifiedContent)
		return
	}

	if cfg.JSONOutput {
		jsonOutput, err := json.Marshal(res)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting result to JSON: %v\n", err)
			return
		}
		fmt.Println(string(jsonOutput))
		return
	}
}

func printFatal(err error, jsonOut bool) {
	if jsonOut {
		if ce, ok := err.(model.CLIError); ok {
			fmt.Println(ce.JSON())
		} else {
			fmt.Println(model.CLIError{
				Code:    model.ErrUnknown,
				Message: err.Error(),
			}.JSON())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func printUsage(fs *pflag.FlagSet) {
	fmt.Fprintf(os.Stderr, "\nUsage: morfx [flags] <file1> <file2> ...\n")
	fmt.Fprintf(os.Stderr, "Quick read usage: morfx ./path/to/file1  //Reads a single file\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	fs.PrintDefaults()
}
