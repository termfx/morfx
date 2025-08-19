package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/evaluator"
	"github.com/termfx/morfx/internal/parser"
	"github.com/termfx/morfx/internal/provider"
	"github.com/termfx/morfx/internal/registry"
)

// CLIConfig holds the configuration parsed from command line arguments
type CLIConfig struct {
	Query     string   // DSL query string
	Files     []string // File patterns to process
	Language  string   // Explicit language selection (-lang flag)
	ShowLangs bool     // Show available languages and exit
	Verbose   bool     // Verbose output
	Help      bool     // Show help and exit
}

// main is the entry point for morfx, implementing the new language-agnostic architecture.
// The CLI workflow follows the specified design:
// 1. Parse command-line arguments
// 2. Initialize registry with built-in providers
// 3. Auto-detect or use specified language
// 4. Get provider from registry
// 5. Parse DSL query using universal parser
// 6. Create universal evaluator with provider
// 7. Evaluate and display results
func main() {
	// Initialize the language provider registry
	if err := initializeRegistry(); err != nil {
		exitWithError(fmt.Errorf("failed to initialize language registry: %w", err))
	}

	// Parse command line arguments
	config, err := parseCommandLine(os.Args[1:])
	if err != nil {
		exitWithError(fmt.Errorf("invalid arguments: %w", err))
	}

	// Handle special flags
	if config.Help {
		showHelp()
		os.Exit(0)
	}

	if config.ShowLangs {
		showAvailableLanguages()
		os.Exit(0)
	}

	// Validate required arguments
	if config.Query == "" {
		exitWithError(fmt.Errorf("DSL query is required"))
	}

	if len(config.Files) == 0 {
		exitWithError(fmt.Errorf("at least one file pattern is required"))
	}

	// Process files and execute queries
	if err := processFiles(config); err != nil {
		exitWithError(fmt.Errorf("processing failed: %w", err))
	}
}

// parseCommandLine parses command line arguments using flag package.
// Supports both POSIX-style flags and the specific morfx usage patterns.
func parseCommandLine(args []string) (*CLIConfig, error) {
	config := &CLIConfig{}

	// Define flags
	flagSet := flag.NewFlagSet("morfx", flag.ContinueOnError)
	flagSet.StringVar(&config.Language, "lang", "", "Explicitly specify the programming language")
	flagSet.StringVar(&config.Language, "language", "", "Explicitly specify the programming language (alias for -lang)")
	flagSet.BoolVar(&config.ShowLangs, "languages", false, "Show available languages and exit")
	flagSet.BoolVar(&config.ShowLangs, "list-languages", false, "Show available languages and exit (alias)")
	flagSet.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flagSet.BoolVar(&config.Verbose, "v", false, "Enable verbose output (short form)")
	flagSet.BoolVar(&config.Help, "help", false, "Show help message")
	flagSet.BoolVar(&config.Help, "h", false, "Show help message (short form)")

	// Suppress flag package output
	flagSet.SetOutput(os.Stderr)

	// Parse flags
	if err := flagSet.Parse(args); err != nil {
		return nil, err
	}

	// Get remaining arguments (query and file patterns)
	remaining := flagSet.Args()

	// Handle different argument patterns:
	// 1. morfx "query" file1 file2 ...
	// 2. morfx -lang=go "query" file1 file2 ...
	if len(remaining) >= 2 {
		config.Query = remaining[0]
		config.Files = remaining[1:]
	} else if len(remaining) == 1 {
		// If only one argument, it could be a query with implicit file pattern
		config.Query = remaining[0]
		// Default to common file patterns if no files specified
		config.Files = []string{"*"}
	}

	return config, nil
}

// processFiles processes each file pattern and executes the query against matching files.
func processFiles(config *CLIConfig) error {
	// Expand file patterns to actual files
	files, err := expandFilePatterns(config.Files)
	if err != nil {
		return fmt.Errorf("failed to expand file patterns: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found matching the patterns: %v", config.Files)
	}

	// Process each file
	totalResults := 0
	for _, file := range files {
		results, err := processFile(file, config)
		if err != nil {
			if config.Verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to process %s: %v\n", file, err)
			}
			continue
		}

		// Display results for this file
		if results.TotalMatches > 0 {
			displayResults(file, results, config)
			totalResults += results.TotalMatches
		}
	}

	// Display summary
	if config.Verbose {
		fmt.Printf("\nTotal matches found: %d across %d files\n", totalResults, len(files))
	}

	return nil
}

// processFile processes a single file with the given query configuration.
func processFile(filename string, config *CLIConfig) (*core.ResultSet, error) {
	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Get language provider
	provider, err := getProviderForFile(filename, config.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get language provider for %s: %w", filename, err)
	}

	if config.Verbose {
		fmt.Printf("Processing %s using %s provider\n", filename, provider.Lang())
	}

	// Parse DSL query using universal parser
	universalParser := parser.NewUniversalParser()
	query, err := universalParser.ParseQuery(config.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSL query '%s': %w", config.Query, err)
	}

	// Create universal evaluator with injected provider
	evaluator, err := evaluator.NewUniversalEvaluator(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluator with %s provider: %w", provider.Lang(), err)
	}

	// Evaluate query against file content
	results, err := evaluator.Evaluate(query, content)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate query against %s: %w", filename, err)
	}

	// Add file information to results
	for _, result := range results.Results {
		result.Location.File = filename
	}

	return results, nil
}

// getProviderForFile determines the appropriate language provider for a file.
// Uses explicit language selection if provided, otherwise auto-detects from file extension.
func getProviderForFile(filename, explicitLang string) (provider.LanguageProvider, error) {
	// Use explicit language if provided
	if explicitLang != "" {
		provider, err := registry.GetProvider(explicitLang)
		if err != nil {
			return nil, fmt.Errorf("language '%s' not supported. Use -languages to see available languages", explicitLang)
		}
		return provider, nil
	}

	// Auto-detect language from file extension
	provider, err := registry.GetProviderForFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot auto-detect language for %s: %w. Use -lang flag to specify explicitly", filename, err)
	}

	return provider, nil
}

// expandFilePatterns expands shell-style patterns to actual file paths.
func expandFilePatterns(patterns []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid file pattern '%s': %w", pattern, err)
		}

		for _, match := range matches {
			// Check if it's a regular file and not already seen
			if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() && !seen[match] {
				files = append(files, match)
				seen[match] = true
			}
		}
	}

	return files, nil
}

// displayResults displays the query results for a file.
func displayResults(filename string, results *core.ResultSet, config *CLIConfig) {
	if results.TotalMatches == 0 {
		return
	}

	// Print file header
	fmt.Printf("\n=== %s ===\n", filename)

	// Print each result
	for i, result := range results.Results {
		fmt.Printf("%d. %s '%s' at line %d:%d\n",
			i+1,
			strings.Title(string(result.Kind)),
			result.Name,
			result.Location.StartLine,
			result.Location.StartCol)

		if config.Verbose {
			// Show additional metadata in verbose mode
			if result.ParentName != "" {
				fmt.Printf("   Parent: %s (%s)\n", result.ParentName, result.ParentKind)
			}
			fmt.Printf("   Scope: %s\n", result.Scope)
			if len(result.Content) > 0 && len(result.Content) < 200 {
				fmt.Printf("   Content: %s\n", strings.TrimSpace(result.Content))
			}
		}
		fmt.Println()
	}

	fmt.Printf("Found %d matches in %s\n", results.TotalMatches, filename)
}

// showAvailableLanguages displays all available language providers.
func showAvailableLanguages() {
	fmt.Println("Available languages:")

	languages := registry.ListProviders()
	if len(languages) == 0 {
		fmt.Println("  No language providers available")
		return
	}

	for _, lang := range languages {
		provider, err := registry.GetProvider(lang)
		if err != nil {
			continue
		}

		aliases := provider.Aliases()
		extensions := provider.Extensions()

		fmt.Printf("  %s", lang)
		if len(aliases) > 1 {
			otherAliases := make([]string, 0)
			for _, alias := range aliases {
				if alias != lang {
					otherAliases = append(otherAliases, alias)
				}
			}
			if len(otherAliases) > 0 {
				fmt.Printf(" (aliases: %s)", strings.Join(otherAliases, ", "))
			}
		}
		if len(extensions) > 0 {
			fmt.Printf(" - extensions: %s", strings.Join(extensions, ", "))
		}
		fmt.Println()
	}
}

// showHelp displays the help message with usage examples.
func showHelp() {
	fmt.Println("morfx - Language-agnostic code query tool")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  morfx [flags] \"DSL_QUERY\" FILE_PATTERNS...")
	fmt.Println()
	fmt.Println("FLAGS:")
	fmt.Println("  -lang LANGUAGE    Explicitly specify the programming language")
	fmt.Println("  -languages        Show available languages and exit")
	fmt.Println("  -verbose, -v      Enable verbose output")
	fmt.Println("  -help, -h         Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Python: Find test functions and User class")
	fmt.Println("  morfx \"def:test* & class:User\" *.py")
	fmt.Println()
	fmt.Println("  # Go: Find Test functions but not mock structs")
	fmt.Println("  morfx \"func:Test* & !struct:mock\" *.go")
	fmt.Println()
	fmt.Println("  # JavaScript: Find test functions or API constants")
	fmt.Println("  morfx \"function:test* | const:API\" *.js")
	fmt.Println()
	fmt.Println("  # Ruby with explicit language specification")
	fmt.Println("  morfx -lang=ruby \"def:initialize & !var:@temp\" *.rb")
	fmt.Println()
	fmt.Println("  # Show available languages")
	fmt.Println("  morfx -languages")
	fmt.Println()
	fmt.Println("DSL QUERY SYNTAX:")
	fmt.Println("  kind:pattern      Match nodes of 'kind' with name matching 'pattern'")
	fmt.Println("  &, &&, and        Logical AND operator")
	fmt.Println("  |, ||, or         Logical OR operator")
	fmt.Println("  !, not            Logical NOT operator")
	fmt.Println("  >                 Hierarchical relationship (parent > child)")
	fmt.Println("  *                 Wildcard matching")
	fmt.Println()
	fmt.Println("SUPPORTED KINDS:")
	fmt.Println("  function, func, def, fn       Functions and methods")
	fmt.Println("  class, struct, type           Classes and types")
	fmt.Println("  variable, var, let            Variables")
	fmt.Println("  constant, const               Constants")
	fmt.Println("  import, require, use          Import statements")
	fmt.Println("  call, invoke                  Function calls")
	fmt.Println("  field, property               Class fields/properties")
	fmt.Println("  ...and many more universal programming concepts")
}

// exitWithError prints an error message and exits with status code 1.
func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	fmt.Fprintf(os.Stderr, "Use 'morfx -help' for usage information.\n")
	os.Exit(1)
}
