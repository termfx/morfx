package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ComponentThresholds defines coverage requirements per component
type ComponentThresholds struct {
	Core      float64 // core/ package
	MCP       float64 // mcp/ package
	Providers float64 // providers/ package
	Safety    float64 // safety-related code
	Database  float64 // db/ and models/ packages
	FileOps   float64 // file processing
	CLI       float64 // cmd/ package
	Utils     float64 // utility packages
}

// Enterprise-level thresholds
var EnterpriseThresholds = ComponentThresholds{
	Core:      85.0, // Critical business logic
	MCP:       80.0, // Protocol implementation
	Providers: 85.0, // Language providers
	Safety:    80.0, // Safety systems
	Database:  70.0, // Database operations
	FileOps:   75.0, // File processing
	CLI:       55.0, // CLI interfaces
	Utils:     65.0, // Utility functions
}

// Strict thresholds for CI/CD
var StrictThresholds = ComponentThresholds{
	Core:      92.0,
	MCP:       87.0,
	Providers: 90.0,
	Safety:    87.0,
	Database:  78.0,
	FileOps:   82.0,
	CLI:       65.0,
	Utils:     72.0,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <coverage.out> [--strict]\n", os.Args[0])
		os.Exit(1)
	}

	coverageFile := os.Args[1]
	strict := len(os.Args) > 2 && os.Args[2] == "--strict"

	thresholds := EnterpriseThresholds
	if strict {
		thresholds = StrictThresholds
		fmt.Println("🔒 Using strict coverage thresholds for CI/CD")
	} else {
		fmt.Println("📊 Using enterprise coverage thresholds")
	}

	coverage, err := parseCoverageFile(coverageFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading coverage file: %v\n", err)
		os.Exit(1)
	}

	componentCoverage := calculateComponentCoverage(coverage)
	overallCoverage := calculateOverallCoverage(coverage)

	fmt.Printf("\n📈 Coverage Report:\n")
	fmt.Printf("Overall: %.1f%%\n\n", overallCoverage)

	failures := 0

	// Check each component
	components := map[string]struct {
		actual    float64
		threshold float64
	}{
		"Core Logic":     {componentCoverage["core"], thresholds.Core},
		"MCP Protocol":   {componentCoverage["mcp"], thresholds.MCP},
		"Providers":      {componentCoverage["providers"], thresholds.Providers},
		"Safety Systems": {componentCoverage["safety"], thresholds.Safety},
		"Database":       {componentCoverage["database"], thresholds.Database},
		"File Ops":       {componentCoverage["fileops"], thresholds.FileOps},
		"CLI":            {componentCoverage["cli"], thresholds.CLI},
		"Utils":          {componentCoverage["utils"], thresholds.Utils},
	}

	for name, data := range components {
		status := "✅"
		if data.actual < data.threshold {
			status = "❌"
			failures++
		}
		fmt.Printf("%s %-15s: %5.1f%% (target: %.1f%%)\n",
			status, name, data.actual, data.threshold)
	}

	// Overall threshold check
	minOverall := 80.0
	if strict {
		minOverall = 82.0
	}

	if overallCoverage < minOverall {
		failures++
		fmt.Printf("❌ Overall coverage %.1f%% below minimum %.1f%%\n", overallCoverage, minOverall)
	}

	if failures > 0 {
		fmt.Printf("\n💥 Coverage check FAILED: %d threshold(s) not met\n", failures)
		fmt.Println("\n🛠️ To improve coverage:")
		fmt.Println("   1. Add unit tests for uncovered code paths")
		fmt.Println("   2. Focus on critical business logic first")
		fmt.Println("   3. Use 'make test-coverage' to see detailed report")
		fmt.Println("   4. Check coverage.html for line-by-line analysis")
		os.Exit(1)
	}

	fmt.Printf("\n🎉 All coverage thresholds met! Great job!\n")
	if overallCoverage >= 85.0 {
		fmt.Println("🏆 Excellent coverage level achieved!")
	}
}

type PackageCoverage struct {
	Package  string
	Coverage float64
	Lines    int
	Covered  int
}

func parseCoverageFile(filename string) ([]PackageCoverage, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var packages []PackageCoverage
	packageMap := make(map[string]*PackageCoverage)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Extract package from filename - use full package path
		fileParts := strings.Split(parts[0], ":")
		if len(fileParts) < 1 {
			continue
		}

		filepath := fileParts[0]
		if filepath == "" {
			continue
		}
		packageName := filepath
		if idx := strings.LastIndex(packageName, "/"); idx != -1 {
			packageName = packageName[:idx]
		} else {
			packageName = strings.TrimSuffix(packageName, ".go")
		}
		packageName = strings.TrimSpace(packageName)
		if packageName == "" {
			continue
		}

		// Parse coverage data
		countStr := parts[len(parts)-1]
		count, err := strconv.Atoi(countStr)
		if err != nil {
			continue
		}

		if _, exists := packageMap[packageName]; !exists {
			packageMap[packageName] = &PackageCoverage{
				Package: packageName,
			}
		}

		pkg := packageMap[packageName]
		pkg.Lines++
		if count > 0 {
			pkg.Covered++
		}
	}

	// Calculate coverage percentages
	for _, pkg := range packageMap {
		if pkg.Lines > 0 {
			pkg.Coverage = float64(pkg.Covered) / float64(pkg.Lines) * 100.0
		}
		packages = append(packages, *pkg)
	}

	return packages, scanner.Err()
}

func calculateComponentCoverage(packages []PackageCoverage) map[string]float64 {
	components := map[string]float64{
		"core":      0.0,
		"mcp":       0.0,
		"providers": 0.0,
		"safety":    0.0,
		"database":  0.0,
		"fileops":   0.0,
		"cli":       0.0,
		"utils":     0.0,
	}

	componentLines := map[string]int{
		"core":      0,
		"mcp":       0,
		"providers": 0,
		"safety":    0,
		"database":  0,
		"fileops":   0,
		"cli":       0,
		"utils":     0,
	}

	componentCovered := map[string]int{
		"core":      0,
		"mcp":       0,
		"providers": 0,
		"safety":    0,
		"database":  0,
		"fileops":   0,
		"cli":       0,
		"utils":     0,
	}

	for _, pkg := range packages {
		var component string

		// Normalize package name for consistent classification
		pkgLower := strings.ToLower(pkg.Package)

		switch {
		case strings.Contains(pkgLower, "core") || strings.Contains(pkgLower, "fileprocessor"):
			component = "core"
		case strings.Contains(pkgLower, "mcp") || strings.Contains(pkgLower, "server"):
			component = "mcp"
		case strings.Contains(pkgLower, "providers") || strings.Contains(pkgLower, "golang") || strings.Contains(pkgLower, "javascript") || strings.Contains(pkgLower, "php") || strings.Contains(pkgLower, "python") || strings.Contains(pkgLower, "typescript"):
			component = "providers"
		case strings.Contains(pkgLower, "safety") || strings.Contains(pkgLower, "atomic") || strings.Contains(pkgLower, "transaction"):
			component = "safety"
		case strings.Contains(pkgLower, "db") || strings.Contains(pkgLower, "models"):
			component = "database"
		case strings.Contains(pkgLower, "filewalker") || strings.Contains(pkgLower, "util"):
			component = "utils"
		case strings.Contains(pkgLower, "file") && !strings.Contains(pkgLower, "core"):
			component = "fileops"
		case strings.Contains(pkgLower, "cmd") || strings.Contains(pkgLower, "main") || strings.Contains(pkgLower, "morfx"):
			component = "cli"
		case strings.Contains(pkgLower, "filewalker") || strings.Contains(pkgLower, "util"):
			component = "utils"
		default:
			component = "utils"
		}

		componentLines[component] += pkg.Lines
		componentCovered[component] += pkg.Covered
	}

	// Calculate percentages
	for component := range components {
		if componentLines[component] > 0 {
			components[component] = float64(componentCovered[component]) / float64(componentLines[component]) * 100.0
		}
	}

	return components
}

func calculateOverallCoverage(packages []PackageCoverage) float64 {
	totalLines := 0
	totalCovered := 0

	for _, pkg := range packages {
		totalLines += pkg.Lines
		totalCovered += pkg.Covered
	}

	if totalLines == 0 {
		return 0.0
	}

	return float64(totalCovered) / float64(totalLines) * 100.0
}
