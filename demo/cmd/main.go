package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/providers/golang"
	"github.com/termfx/morfx/providers/javascript"
	"github.com/termfx/morfx/providers/php"
	"github.com/termfx/morfx/providers/typescript"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	blue   = color.New(color.FgBlue).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
)

type DemoRunner struct {
	workDir     string
	backupDir   string
	fixturesDir string
	providers   map[string]any // provider instances
}

func NewDemoRunner() *DemoRunner {
	wd, _ := os.Getwd()
	return &DemoRunner{
		workDir:     filepath.Join(wd, "demo"),
		backupDir:   filepath.Join(wd, "demo", ".demo-backup"),
		fixturesDir: filepath.Join(wd, "demo", "fixtures"),
		providers: map[string]any{
			"go":         golang.New(),
			"php":        php.New(),
			"javascript": javascript.New(),
			"typescript": typescript.New(),
		},
	}
}

func (d *DemoRunner) CreateBackup() error {
	fmt.Printf("%s Creating backup...\n", yellow("📦"))
	os.RemoveAll(d.backupDir)

	if err := os.MkdirAll(d.backupDir, 0o755); err != nil {
		return fmt.Errorf("failed to create backup dir: %w", err)
	}

	return copyDir(d.fixturesDir, d.backupDir)
}

func (d *DemoRunner) RestoreBackup() error {
	fmt.Printf("%s Restoring original files...\n", blue("🔄"))
	os.RemoveAll(d.fixturesDir)
	return copyDir(d.backupDir, d.fixturesDir)
}

func (d *DemoRunner) RunDemo(scenario string) error {
	fmt.Printf("%s %s\n", bold("🚀"), bold("Morfx AST Transformation Demo"))
	fmt.Println(strings.Repeat("═", 60))

	if err := d.CreateBackup(); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	defer func() {
		time.Sleep(2 * time.Second)
		d.RestoreBackup()
		fmt.Printf("%s Demo completed - files restored\n", green("✅"))
	}()

	scenarios := d.getScenarios()

	// Run specific scenario or all
	if scenario != "" && scenario != "all" {
		if s, exists := scenarios[scenario]; exists {
			return d.runScenario(scenario, s)
		}
		return fmt.Errorf("scenario %q not found", scenario)
	}

	// Run all scenarios in order
	order := []string{"go-query", "php-replace", "js-insert", "ts-delete"}
	for _, name := range order {
		if s, exists := scenarios[name]; exists {
			if err := d.runScenario(name, s); err != nil {
				fmt.Printf("%s Error: %v\n", red("❌"), err)
			}
			fmt.Println()
			time.Sleep(1 * time.Second)
		}
	}

	return nil
}

func (d *DemoRunner) runScenario(_ string, scenario Scenario) error {
	fmt.Printf("\n%s %s\n", cyan("▶"), bold(scenario.Description))
	fmt.Println(strings.Repeat("─", 60))

	// Show before state
	fmt.Printf("\n%s Before:\n", blue("📄"))
	beforeContent, err := os.ReadFile(scenario.File)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	d.showFileContent(scenario.File, 10)

	// Execute the transformation
	fmt.Printf("\n%s Executing transformation...\n", yellow("⚡"))

	result, err := d.executeTransformation(scenario, string(beforeContent))
	if err != nil {
		return fmt.Errorf("transformation failed: %w", err)
	}

	// Show results based on operation type
	switch scenario.Operation {
	case "query":
		d.showQueryResults(result)

	case "replace", "delete", "insert_after", "insert_before", "append":
		// Write the modified content back
		if modified, ok := result["modified"].(string); ok {
			if err := os.WriteFile(scenario.File, []byte(modified), 0o644); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}

			// Show diff
			d.showDiff(string(beforeContent), modified)

			// Show after state
			fmt.Printf("\n%s After:\n", green("📝"))
			d.showFileContent(scenario.File, 10)
		}
	}

	return nil
}

func (d *DemoRunner) executeTransformation(scenario Scenario, source string) (map[string]any, error) {
	provider := d.providers[scenario.Language]
	if provider == nil {
		return nil, fmt.Errorf("no provider for language %s", scenario.Language)
	}

	switch scenario.Operation {
	case "query":
		if p, ok := provider.(interface {
			Query(string, core.AgentQuery) core.QueryResult
		}); ok {
			result := p.Query(source, scenario.Query)
			return map[string]any{
				"matches": result.Matches,
				"error":   result.Error,
			}, nil
		}

	case "replace", "delete", "insert_after", "insert_before", "append":
		if p, ok := provider.(interface {
			Transform(string, core.TransformOp) core.TransformResult
		}); ok {
			op := core.TransformOp{
				Method:      scenario.Operation,
				Target:      scenario.Query,
				Replacement: scenario.Replacement,
				Content:     scenario.Content,
			}
			result := p.Transform(source, op)
			if result.Error != nil {
				return nil, result.Error
			}
			return map[string]any{
				"modified":   result.Modified,
				"confidence": result.Confidence,
			}, nil
		}
	}

	return nil, fmt.Errorf("operation %s not supported", scenario.Operation)
}

func (d *DemoRunner) showQueryResults(result map[string]any) {
	fmt.Printf("\n%s Results:\n", green("🔍"))

	if matches, ok := result["matches"].([]core.Match); ok {
		if len(matches) == 0 {
			fmt.Printf("  %s No matches found\n", yellow("→"))
		} else {
			fmt.Printf("  %s Found %d matches:\n", green("✓"), len(matches))
			for i, m := range matches {
				fmt.Printf("  %d. %s '%s' at line %d\n",
					i+1, m.Type, m.Name, m.Location.Line+1)
				// Show first few lines of content
				lines := strings.Split(m.Content, "\n")
				for j, line := range lines {
					if j >= 3 {
						fmt.Printf("     %s\n", yellow("..."))
						break
					}
					fmt.Printf("     %s\n", strings.TrimSpace(line))
				}
			}
		}
	}
}

func (d *DemoRunner) showDiff(before, after string) {
	fmt.Printf("\n%s Changes:\n", green("📝"))

	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")

	// Simple diff - just show a few changed lines
	changed := false
	for i := 0; i < len(beforeLines) && i < len(afterLines); i++ {
		if beforeLines[i] != afterLines[i] {
			if !changed {
				changed = true
				if i > 0 {
					fmt.Printf("  %s\n", beforeLines[i-1])
				}
			}
			fmt.Printf("  %s\n", red("- "+beforeLines[i]))
			fmt.Printf("  %s\n", green("+ "+afterLines[i]))
			if i+1 < len(beforeLines) {
				fmt.Printf("  %s\n", beforeLines[i+1])
			}
			break
		}
	}

	if !changed {
		fmt.Printf("  %s No visible changes\n", yellow("→"))
	}
}

type Scenario struct {
	Description string
	File        string
	Language    string
	Operation   string // query, replace, delete, insert_after, etc.
	Query       core.AgentQuery
	Replacement string // for replace
	Content     string // for insert operations
	Command     string // display only
}

func (d *DemoRunner) getScenarios() map[string]Scenario {
	return map[string]Scenario{
		"go-query": {
			Description: "Go: Query functions with 'User' in name",
			File:        filepath.Join(d.fixturesDir, "example.go"),
			Language:    "go",
			Operation:   "query",
			Query: core.AgentQuery{
				Type: "function",
				Name: "*User*",
			},
			Command: "morfx:query",
		},
		"php-replace": {
			Description: "PHP: Replace updateEmail method with validation",
			File:        filepath.Join(d.fixturesDir, "example.php"),
			Language:    "php",
			Operation:   "replace",
			Query: core.AgentQuery{
				Type: "method",
				Name: "updateEmail",
			},
			Replacement: `    public function updateEmail($newEmail) {
        if (!filter_var($newEmail, FILTER_VALIDATE_EMAIL)) {
            throw new InvalidArgumentException("Invalid email format");
        }
        $this->email = $newEmail;
        $this->updatedAt = new DateTime();
        return $this;
    }`,
			Command: "morfx:replace",
		},
		"js-insert": {
			Description: "JavaScript: Insert phone validation after email validation",
			File:        filepath.Join(d.fixturesDir, "example.js"),
			Language:    "javascript",
			Operation:   "insert_after",
			Query: core.AgentQuery{
				Type: "function",
				Name: "validateEmail",
			},
			Content: `
function validatePhone(phone) {
    const phoneRegex = /^\+?[\d\s\-\(\)]+$/;
    return phoneRegex.test(phone) && phone.length >= 10;
}`,
			Command: "morfx:insert_after",
		},
		"ts-delete": {
			Description: "TypeScript: Delete createUser function",
			File:        filepath.Join(d.fixturesDir, "example.ts"),
			Language:    "typescript",
			Operation:   "delete",
			Query: core.AgentQuery{
				Type: "function",
				Name: "createUser",
			},
			Command: "morfx:delete",
		},
	}
}

func (d *DemoRunner) showFileContent(file string, lines int) {
	content, err := os.ReadFile(file)
	if err != nil {
		fmt.Printf("  %s Error reading %s: %v\n", red("✗"), file, err)
		return
	}

	fmt.Printf("  %s %s (%d bytes)\n", yellow("→"), filepath.Base(file), len(content))

	fileLines := strings.Split(string(content), "\n")
	for i, line := range fileLines {
		if i >= lines {
			fmt.Printf("  %s ... (%d more lines)\n", yellow("→"), len(fileLines)-i)
			break
		}
		fmt.Printf("  %2d | %s\n", i+1, line)
	}
}

// Helper functions
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "demo",
		Short: "Morfx AST transformation demo",
		Long:  "Interactive demonstration of Morfx capabilities with real code transformations",
	}

	runCmd := &cobra.Command{
		Use:   "run [scenario]",
		Short: "Run demo scenarios",
		Long:  "Run transformation demo scenarios. Leave empty for all scenarios.",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			scenario := ""
			if len(args) > 0 {
				scenario = args[0]
			}

			runner := NewDemoRunner()
			if err := runner.RunDemo(scenario); err != nil {
				fmt.Printf("%s %v\n", red("Error:"), err)
				os.Exit(1)
			}
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available scenarios",
		Run: func(cmd *cobra.Command, args []string) {
			runner := NewDemoRunner()
			scenarios := runner.getScenarios()

			fmt.Printf("\n%s Available Demo Scenarios\n", bold("📚"))
			fmt.Println(strings.Repeat("═", 60))

			for name, scenario := range scenarios {
				fmt.Printf("\n%s %s\n", cyan("•"), bold(name))
				fmt.Printf("  %s\n", scenario.Description)
				fmt.Printf("  Language: %s\n", scenario.Language)
				fmt.Printf("  Operation: %s\n", scenario.Operation)
			}
			fmt.Println()
		},
	}

	rootCmd.AddCommand(runCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
