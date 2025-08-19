package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ValidationResult represents the result of a single validation test
type ValidationResult struct {
	Name        string
	Description string
	Passed      bool
	Details     []string
	Issues      []string
}

// ValidationReport represents the complete validation report
type ValidationReport struct {
	Results []ValidationResult
	Summary ValidationSummary
}

// ValidationSummary provides an overall summary of validation results
type ValidationSummary struct {
	TotalTests  int
	PassedTests int
	FailedTests int
	PassRate    float64
}

func main() {
	fmt.Println("🔍 MORFX Architecture Validation")
	fmt.Println("==================================")
	fmt.Println()

	report := &ValidationReport{}

	// Run all validation tests
	report.Results = append(report.Results, validateZeroImports())
	report.Results = append(report.Results, validateProviderIsolation())
	report.Results = append(report.Results, validateSingleEvaluator())
	report.Results = append(report.Results, validateCleanBoundaries())

	// Calculate summary
	report.Summary = calculateSummary(report.Results)

	// Print results
	printReport(report)

	// Exit with appropriate code
	if report.Summary.PassedTests == report.Summary.TotalTests {
		fmt.Println("✅ All architecture validation tests PASSED!")
		os.Exit(0)
	} else {
		fmt.Printf("❌ %d of %d tests FAILED!\n", report.Summary.FailedTests, report.Summary.TotalTests)
		os.Exit(1)
	}
}

// validateZeroImports validates that core modules don't import any internal/lang/* packages
func validateZeroImports() ValidationResult {
	result := ValidationResult{
		Name:        "Zero Imports",
		Description: "Core modules must not import any internal/lang/* packages",
		Details:     []string{},
		Issues:      []string{},
	}

	coreModules := []string{
		"internal/core",
		"internal/parser",
		"internal/evaluator",
		"internal/registry",
		"internal/provider",
	}

	for _, module := range coreModules {
		err := checkModuleImports(module, &result)
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("Failed to check %s: %v", module, err))
		}
	}

	result.Passed = len(result.Issues) == 0
	return result
}

// checkModuleImports checks if a module imports any internal/lang/* packages
func checkModuleImports(modulePath string, result *ValidationResult) error {
	return filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(importPath, "/internal/lang/") {
				result.Issues = append(result.Issues,
					fmt.Sprintf("%s imports forbidden package: %s", path, importPath))
			} else {
				result.Details = append(result.Details,
					fmt.Sprintf("%s: ✓ Clean imports", path))
			}
		}

		return nil
	})
}

// validateProviderIsolation validates that providers don't import each other
func validateProviderIsolation() ValidationResult {
	result := ValidationResult{
		Name:        "Provider Isolation",
		Description: "Language providers must not import each other",
		Details:     []string{},
		Issues:      []string{},
	}

	providerDirs := []string{
		"internal/lang/golang",
		"internal/lang/python",
		"internal/lang/javascript",
		"internal/lang/typescript",
	}

	for _, providerDir := range providerDirs {
		err := checkProviderImports(providerDir, providerDirs, &result)
		if err != nil {
			result.Issues = append(result.Issues, fmt.Sprintf("Failed to check %s: %v", providerDir, err))
		}
	}

	result.Passed = len(result.Issues) == 0
	return result
}

// checkProviderImports checks if a provider imports other providers
func checkProviderImports(providerPath string, allProviders []string, result *ValidationResult) error {
	return filepath.Walk(providerPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Check if this import is to another provider
			for _, otherProvider := range allProviders {
				if otherProvider != providerPath && strings.Contains(importPath, otherProvider) {
					result.Issues = append(result.Issues,
						fmt.Sprintf("%s imports other provider: %s", path, importPath))
				}
			}
		}

		result.Details = append(result.Details, fmt.Sprintf("%s: ✓ No cross-provider imports", path))
		return nil
	})
}

// validateSingleEvaluator validates that there's only one evaluator implementation
func validateSingleEvaluator() ValidationResult {
	result := ValidationResult{
		Name:        "Single Evaluator",
		Description: "Must have exactly one universal evaluator implementation",
		Details:     []string{},
		Issues:      []string{},
	}

	evaluatorPath := "internal/evaluator"
	evaluatorCount := 0

	err := filepath.Walk(evaluatorPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Look for evaluator struct definitions
		ast.Inspect(node, func(n ast.Node) bool {
			if typeSpec, ok := n.(*ast.TypeSpec); ok {
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					typeName := typeSpec.Name.Name
					if strings.Contains(strings.ToLower(typeName), "evaluator") {
						evaluatorCount++
						result.Details = append(result.Details,
							fmt.Sprintf("Found evaluator: %s in %s", typeName, path))
					}
					_ = structType // Use the variable
				}
			}
			return true
		})

		return nil
	})
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to analyze evaluator: %v", err))
	}

	if evaluatorCount != 1 {
		result.Issues = append(result.Issues,
			fmt.Sprintf("Expected 1 evaluator implementation, found %d", evaluatorCount))
	} else {
		result.Details = append(result.Details, "✓ Exactly one universal evaluator found")
	}

	result.Passed = len(result.Issues) == 0
	return result
}

// validateCleanBoundaries validates that interfaces between layers are properly defined
func validateCleanBoundaries() ValidationResult {
	result := ValidationResult{
		Name:        "Clean Boundaries",
		Description: "Interfaces between layers must be properly defined",
		Details:     []string{},
		Issues:      []string{},
	}

	// Check that the LanguageProvider interface exists and is complete
	providerPath := "internal/provider/contract.go"
	err := checkProviderInterface(providerPath, &result)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to check provider interface: %v", err))
	}

	result.Passed = len(result.Issues) == 0
	return result
}

// checkProviderInterface verifies that the LanguageProvider interface is properly defined
func checkProviderInterface(path string, result *ValidationResult) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	var foundInterface bool
	var methodCount int

	ast.Inspect(node, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				if typeSpec.Name.Name == "LanguageProvider" {
					foundInterface = true
					methodCount = len(interfaceType.Methods.List)
					result.Details = append(result.Details,
						fmt.Sprintf("✓ LanguageProvider interface found with %d methods", methodCount))
				}
			}
		}
		return true
	})

	if !foundInterface {
		result.Issues = append(result.Issues, "LanguageProvider interface not found")
	}

	if methodCount < 10 { // Expect at least 10 core methods
		result.Issues = append(result.Issues,
			fmt.Sprintf("LanguageProvider interface has insufficient methods: %d", methodCount))
	}

	return nil
}

// calculateSummary calculates validation summary statistics
func calculateSummary(results []ValidationResult) ValidationSummary {
	total := len(results)
	passed := 0

	for _, result := range results {
		if result.Passed {
			passed++
		}
	}

	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100.0
	}

	return ValidationSummary{
		TotalTests:  total,
		PassedTests: passed,
		FailedTests: total - passed,
		PassRate:    passRate,
	}
}

// printReport prints the validation report
func printReport(report *ValidationReport) {
	for i, result := range report.Results {
		fmt.Printf("%d. %s\n", i+1, result.Name)
		fmt.Printf("   Description: %s\n", result.Description)

		if result.Passed {
			fmt.Printf("   Status: ✅ PASSED\n")
		} else {
			fmt.Printf("   Status: ❌ FAILED\n")
		}

		if len(result.Details) > 0 {
			fmt.Printf("   Details:\n")
			for _, detail := range result.Details {
				if len(detail) > 80 {
					detail = detail[:77] + "..."
				}
				fmt.Printf("     • %s\n", detail)
			}
		}

		if len(result.Issues) > 0 {
			fmt.Printf("   Issues:\n")
			for _, issue := range result.Issues {
				fmt.Printf("     ❌ %s\n", issue)
			}
		}

		fmt.Println()
	}

	fmt.Println("SUMMARY")
	fmt.Println("=======")
	fmt.Printf("Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Printf("Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("Failed: %d\n", report.Summary.FailedTests)
	fmt.Printf("Pass Rate: %.1f%%\n", report.Summary.PassRate)
	fmt.Println()
}
