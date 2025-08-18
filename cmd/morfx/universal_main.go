package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/garaekz/fileman/internal/evaluator"
	"github.com/garaekz/fileman/internal/lang/golang"
	"github.com/garaekz/fileman/internal/lang/javascript"
	"github.com/garaekz/fileman/internal/lang/python"
	"github.com/garaekz/fileman/internal/lang/typescript"
	"github.com/garaekz/fileman/internal/parser"
	"github.com/garaekz/fileman/internal/registry"
	"github.com/garaekz/fileman/internal/types"
)

func universalMain() {
	var lang string
	var pattern string
	var scope string

	flag.StringVar(&lang, "lang", "", "Language (auto-detect if empty)")
	flag.StringVar(&pattern, "pattern", "", "DSL pattern")
	flag.StringVar(&scope, "scope", "file", "Scope for boolean operations")
	flag.Parse()

	// Register all built-in providers
	registry.Register(golang.NewProvider())
	registry.Register(python.NewProvider())
	registry.Register(javascript.NewProvider())
	registry.Register(typescript.NewProvider())

	// Also try loading external plugins
	registry.AutoRegister()

	// Auto-detect language from file extension if not specified
	if lang == "" {
		lang = detectLanguage(flag.Args())
	}

	// Get provider
	provider, err := registry.GetProvider(lang)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Available languages:")
		for _, p := range registry.ListProviders() {
			fmt.Printf("  - %s\n", p)
		}
		os.Exit(1)
	}

	// Parse DSL (universal parser)
	p := parser.NewUniversalParser()
	query, err := p.ParseQuery(pattern)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		os.Exit(1)
	}

	// Set scope
	query.Scope = types.ScopeType(scope)

	// Create universal evaluator with language-specific provider
	eval, err := evaluator.NewUniversalEvaluator(provider)
	if err != nil {
		fmt.Printf("Failed to create evaluator: %v\n", err)
		os.Exit(1)
	}

	// Process files
	for _, file := range flag.Args() {
		source, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", file, err)
			continue
		}

		results, err := eval.EvaluateQuery(query, source)
		if err != nil {
			fmt.Printf("Error in %s: %v\n", file, err)
			continue
		}

		// Display results
		for _, result := range results.All() {
			fmt.Printf("%s:%d:%d %s %s\n",
				file,
				result.Location.StartLine,
				result.Location.StartCol,
				result.Kind,
				result.Name,
			)
		}
	}
}

func detectLanguage(files []string) string {
	if len(files) == 0 {
		return ""
	}

	ext := filepath.Ext(files[0])

	// Common mappings
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".php":
		return "php"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	default:
		// Try using registry to detect by extension
		provider, err := registry.GetProviderByExtension(ext)
		if err == nil {
			return provider.Lang()
		}
		return ""
	}
}

// Example usage function showing cross-language queries
func showExamples() {
	fmt.Println("MORFX Universal CLI - Language-Agnostic Code Query Tool")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println()
	fmt.Println("  # Same query, different languages - it just works!")
	fmt.Println()
	fmt.Println("  # Go")
	fmt.Println("  morfx -lang=go \"function:Test* && !variable:err\" *.go")
	fmt.Println()
	fmt.Println("  # Python")
	fmt.Println("  morfx -lang=python \"function:test* && !variable:err\" *.py")
	fmt.Println()
	fmt.Println("  # JavaScript")
	fmt.Println("  morfx -lang=js \"function:test* && !variable:err\" *.js")
	fmt.Println()
	fmt.Println("  # TypeScript")
	fmt.Println("  morfx -lang=ts \"function:test* && !variable:err\" *.ts")
	fmt.Println()
	fmt.Println("  # Auto-detect language from file extension")
	fmt.Println("  morfx \"class:*Controller > method:handle*\" src/**/*.php")
	fmt.Println()
	fmt.Println("  # Hierarchical queries")
	fmt.Println("  morfx \"class:User > method:get*\" *.java")
	fmt.Println()
	fmt.Println("  # Complex boolean operations")
	fmt.Println("  morfx \"(function:parse* || function:compile*) && !call:panic\" *.go")
	fmt.Println()
	fmt.Println("Supported languages:")
	for _, lang := range registry.ListProviders() {
		fmt.Printf("  - %s\n", lang)
	}
}
