package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/garaekz/fileman/internal/cli"
	"github.com/garaekz/fileman/internal/lang"
	_ "github.com/garaekz/fileman/internal/lang/golang" // Register the Go provider
	"github.com/garaekz/fileman/internal/model"
	"github.com/garaekz/fileman/internal/util"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "ast":
		handleASTCommand()
	// case "regex":
	// 	handleRegexCommand() // Future: move regex flags here
	default:
		// TODO: Remove this once all commands are migrated
		handleDefaultCommand()
	}
}

func handleASTCommand() {
	astCmd := flag.NewFlagSet("ast", flag.ExitOnError)

	// --- Flag Definitions for 'ast' subcommand ---
	var inputFiles multiFlag
	target := astCmd.String("target", "", `Semantic target, format: "type:name" (e.g., "function:MyFunc") (required)`)
	langFlag := astCmd.String("lang", "go", "Language for AST matching (go, php, etc.)")
	operation := astCmd.String("operation", "get", "Operation: get|replace|insert-before|insert-after|delete.")
	replacement := astCmd.String("replacement", "", "Replacement string (from flag). Overridden by stdin.")
	dedupe := astCmd.Bool("dedupe", true, "Enable auto-deduplication on insert operations.")

	// Behavior and Output flags
	dryRun := astCmd.Bool("dry-run", false, "Perform a trial run without writing any files.")
	verbose := astCmd.Bool("verbose", false, "Enable verbose output.")
	jsonOutput := astCmd.Bool("json", false, "Output results in JSON format.")
	stdoutMode := astCmd.Bool("stdout", false, "Print final content to stdout (implied by op 'get').")
	showDiff := astCmd.Bool("diff", false, "Show a unified diff of the changes.")
	diffContext := astCmd.Int("diff-context", 3, "Lines of context for the diff.")
	colorDiff := astCmd.Bool("color", true, "Colorize diff output.")
	workers := astCmd.Int("workers", 0, "Number of parallel workers (0 for auto).")

	astCmd.Var(&inputFiles, "file", "File(s) to process (repeatable), use '-' for stdin.")

	astCmd.Parse(os.Args[2:])

	// --- Validation and Config Building ---
	if *target == "" {
		fmt.Fprintln(os.Stderr, "Error: --target flag is required for the ast command.")
		astCmd.Usage()
		os.Exit(2)
	}
	if len(inputFiles) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one -file is required.")
		astCmd.Usage()
		os.Exit(2)
	}

	// --- Read replacement from stdin if available ---
	finalReplacement := *replacement
	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking stdin: %v\n", err)
		os.Exit(1)
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		if len(stdinBytes) > 0 {
			finalReplacement = string(stdinBytes)
		}
	}

	parts := strings.SplitN(*target, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		fmt.Fprintf(os.Stderr, "Error: invalid --target format. Expected \"type:name\".\n")
		os.Exit(2)
	}

	query, err := lang.GetQueryForLanguage(*langFlag, parts[0], parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := model.ModificationConfig{
		RuleID:      fmt.Sprintf("ast-%s-%s", *operation, *target),
		Pattern:     query,
		Replacement: finalReplacement,
		Operation:   model.Operation(*operation),
		UseAST:      true,
		Lang:        *langFlag,
		Dedupe:      *dedupe,
	}

	runner := &cli.Runner{
		DryRun:      *dryRun,
		Verbose:     *verbose,
		JSONOutput:  *jsonOutput,
		StdoutMode:  *stdoutMode || (*operation == "get"),
		ShowDiff:    *showDiff,
		DiffContext: *diffContext,
		ColorDiff:   *colorDiff,
		Workers:     *workers,
	}

	// --- Execution ---
	files := util.ExpandGlobs(inputFiles)
	exitCode := runner.RunWithFlags(files, cfg, false)
	os.Exit(exitCode)
}

// handleDefaultCommand contains the original flag logic for backward compatibility
// or for the primary regex-based operations. It should be refactored as needed.
func handleDefaultCommand() {
	var (
		// Mode flags
		configFile = flag.String("config", "", "Path to a JSON configuration file for multi-rule processing.")
		inputFiles multiFlag

		// Single-rule config flags
		pattern             = flag.String("pattern", "", "Regular expression pattern.")
		patternFile         = flag.String("pattern-file", "", "Read pattern from file instead of --pattern.")
		literalPattern      = flag.Bool("literal-pattern", false, "Treat the pattern as a literal string.")
		normalizeWhitespace = flag.Bool("normalize-whitespace", false, "Normalize all whitespace before matching.")
		useAST              = flag.Bool("use-ast", false, "Use AST-based structural matching instead of regex.")
		lang                = flag.String("lang", "go", "Language for AST matching (e.g. go, python).")
		dedupe              = flag.Bool("dedupe", true, "Enable auto-deduplication on insert operations.")
		replacement         = flag.String("replacement", "", "Replacement string for replace/insert operations.")
		operation           = flag.String("operation", "replace", "Operation: replace|insert-before|insert-after|delete.")
		occurrences         = flag.String("occurrences", "all", "Occurrences to modify: first|all|<n>.")
		ruleID              = flag.String("rule-id", "cli-rule", "Identifier for the rule in single-rule mode.")
		mustMatch           = flag.Int("must-match", 0, "Require at least N matches for the rule to succeed.")
		mustChange          = flag.Int("must-change-bytes", 0, "Require at least N bytes changed for the rule to succeed.")

		// Behavior flags
		dryRun        = flag.Bool("dry-run", false, "Perform a trial run without writing any files.")
		failIfNoMatch = flag.Bool("fail-if-no-match", false, "Exit with an error code if no matches are found.")
		stdinMode     = flag.Bool("stdin", false, "Read content from stdin (equivalent to -file -).")

		// Output flags
		verbose     = flag.Bool("verbose", false, "Enable verbose output.")
		jsonOutput  = flag.Bool("json", false, "Output results in JSON format.")
		stdoutMode  = flag.Bool("stdout", false, "Print the final modified content to stdout.")
		showDiff    = flag.Bool("diff", false, "Show a unified diff of the changes.")
		diffContext = flag.Int("diff-context", 3, "Number of context lines for the unified diff.")
		colorDiff   = flag.Bool("color", true, "Colorize the diff output.")
	)
	flag.Var(&inputFiles, "file", "File(s) to process (repeatable), use '-' for stdin.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Select pattern: file overrides inline
	finalPattern := *pattern
	if *patternFile != "" {
		data, err := os.ReadFile(*patternFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading pattern file: %v\n", err)
			os.Exit(1)
		}
		finalPattern = string(data)
	}

	runner := &cli.Runner{
		DryRun:      *dryRun,
		Verbose:     *verbose,
		JSONOutput:  *jsonOutput,
		StdoutMode:  *stdoutMode,
		ShowDiff:    *showDiff,
		DiffContext: *diffContext,
		ColorDiff:   *colorDiff,
	}

	var exitCode int
	if *configFile != "" {
		if *pattern != "" || *replacement != "" || *operation != "replace" ||
			*occurrences != "all" ||
			*ruleID != "cli-rule" ||
			*mustMatch != 0 ||
			*mustChange != 0 ||
			*literalPattern ||
			*normalizeWhitespace {
			fmt.Fprintln(os.Stderr, "Error: Cannot use --config with single-rule flags.")
			os.Exit(2)
		}
		exitCode = runner.RunWithConfig(*configFile)
	} else {
		if *stdinMode {
			inputFiles = append(inputFiles, "-")
		}
		if len(inputFiles) == 0 {
			fmt.Fprintln(os.Stderr, "Error: At least one -file or -stdin is required.")
			flag.Usage()
			os.Exit(2)
		}

		// Single-rule mode
		cfg := model.ModificationConfig{
			RuleID:              *ruleID,
			Pattern:             finalPattern,
			Replacement:         *replacement,
			Operation:           model.Operation(*operation),
			Occurrences:         *occurrences,
			MustMatch:           *mustMatch,
			MustChangeBytes:     *mustChange,
			NormalizeWhitespace: *normalizeWhitespace,
			LiteralPattern:      *literalPattern,
			UseAST:              *useAST,
			Lang:                *lang,
			Dedupe:              *dedupe,
		}

		files := util.ExpandGlobs(inputFiles)
		exitCode = runner.RunWithFlags(files, cfg, *failIfNoMatch)
	}
	os.Exit(exitCode)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  ast      Perform structural, AST-based code manipulation.")
	// Add other commands here
	fmt.Fprintln(os.Stderr, "\nRun 'fileman <command> --help' for more information.")
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
