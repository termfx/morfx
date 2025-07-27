package main

import (
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
	"github.com/garaekz/fileman/internal/util"
)

// main is the entry point for morfx, the command-line tool for file transformations.
// It parses command-line flags, builds a configuration, and runs the transformation.
func main() {
	cfg, files, err := buildConfigFromFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	runner := &cli.Runner{}
	var res []model.Result
	if cfg.Operation == model.OpGet && cfg.RuleID == "cli-first-arg" {
		// Handle a special case to run it Harmlessly
		res, err = runner.RunHarmless(files[0], cfg)
	}
	// else {
	// 	// Normal run with the provided files and configuration
	// 	res = runner.Run(files, cfg)
	// }
	handleOutputAndExit(res, err, cfg)
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
	showDiff := fs.BoolP("diff", "D", false, "Show a unified diff of the changes.")
	diffContext := fs.IntP("diff-context", "C", 3, "Lines of context for the diff.")
	verbose := fs.BoolP("verbose", "v", false, "Enable verbose output.")
	jsonOutput := fs.BoolP("json", "j", false, "Output results in JSON format.")
	stdout := fs.Bool("stdout", false, "Output modified content to stdout instead of writing to files.")

	// Parse the flags
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	// If the --help flag is set, show usage and return an error
	if !fs.HasFlags() || fs.Changed("help") {
		fs.Usage()
		return nil, nil, flag.ErrHelp
	}

	// If only one argument is provided we take a shortcut
	// and try to resolve the language provider based on the file extension.
	if fs.NArg() == 1 {
		if _, err := os.Stat(fs.Arg(0)); os.IsNotExist(err) {
			return nil, nil, errors.New("the specified file does not exist")
		}

		if filepath.Ext(fs.Arg(0)) == "" {
			return nil, nil, errors.New("the first argument must be a file path")
		}

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

	// If no query is provided
	if *query == "" {
		return nil, nil, errors.New("the --query flag is required")
	}

	// File paths are expected as positional arguments after the flags
	files := fs.Args()
	if len(files) == 0 {
		return nil, nil, errors.New("at least one file path is required")
	}

	provider, err := lang.ResolveProvider(*langFlag, files)
	if err != nil {
		return nil, nil, err
	}

	tsQuery := *query

	cfg := &model.Config{
		RuleID:        "cli-operation",
		Pattern:       tsQuery,
		Replacement:   *replacement,
		Operation:     model.Operation(*operation),
		Provider:      provider,
		DryRun:        *dryRun,
		ShowDiff:      *showDiff,
		DiffContext:   *diffContext,
		Verbose:       *verbose,
		JSONOutput:    *jsonOutput,
		StdoutMode:    *stdout,
		FailIfNoMatch: !*force,
		Workers:       *workers,
	}

	if !*includeTests {
		ignorePatterns, _ := provider.GetDefaultIgnorePatterns()
		files = filterFiles(files, ignorePatterns)
	}

	const diffThreshold = 50
	if len(files) > diffThreshold && !*dryRun {
		fmt.Fprintf(os.Stdout, "Warning: This operation affects %d files. Forcing diff preview.\n", len(files))
		cfg.ShowDiff = true
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
		if len(stdinBytes) > 0 {
			cfg.Replacement = string(stdinBytes)
		}
	}

	return cfg, util.ExpandGlobs(files), nil
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

func handleOutputAndExit(res []model.Result, err error, cfg *model.Config) {
	if err != nil {
		printFatal(err, cfg.JSONOutput)
	}
	for _, r := range res {
		printResultCLI(&r, cfg)
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
