package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/internal/toolenv"
)

const recipeHelp = `Usage: recipe [-h]

Reads a Morfx recipe JSON document from stdin and emits a JSON response to stdout.

Input schema:
{
  "name": "repeatable transform name",
  "description": "optional description",
  "dry_run": true,
  "min_confidence": 0.85,
  "steps": [
    {
      "name": "replace one target family",
      "method": "replace",
      "scope": {
        "path": ".",
        "include": ["**/*.go"],
        "exclude": ["vendor/**"],
        "language": "go"
      },
      "target": {"type": "function", "name": "Old*"},
      "target_dsl": "func:Old*",
      "replacement": "func NewName() {}"
    }
  ]
}

Supported step methods: replace, delete, insert_before, insert_after, append.
Apply-mode recipes always run a dry-run preflight first and only mutate files
when each step meets its min_confidence gate.
`

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Usage = func() {
		fmt.Print(recipeHelp)
	}
	flag.Parse()
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	env, err := toolenv.NewEnvironment()
	if err != nil {
		writeErrorAndExit("failed to initialise environment", err)
	}

	req, err := toolenv.ReadJSON[core.Recipe](os.Stdin)
	if err != nil {
		writeErrorAndExit("invalid input", err)
	}

	if err := normalizeRecipeScopes(req); err != nil {
		writeErrorAndExit("invalid recipe scope", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := core.ExecuteRecipe(ctx, env.FileProcessor(), *req)
	if err != nil {
		writeErrorAndExit("recipe failed", err)
	}

	payload := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": formatRecipeResponse(result),
		}},
		"name":            result.Name,
		"dry_run":         result.DryRun,
		"steps_run":       result.StepsRun,
		"files_scanned":   result.FilesScanned,
		"files_modified":  result.FilesModified,
		"matches":         result.TotalMatches,
		"transaction_ids": result.TransactionIDs,
		"steps":           result.Steps,
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func writeErrorAndExit(message string, err error) {
	if writeErr := toolenv.WriteError(os.Stdout, message, err); writeErr != nil {
		fmt.Fprintf(os.Stderr, "failed to write error output: %v\n", writeErr)
	}
	os.Exit(1)
}

func normalizeRecipeScopes(recipe *core.Recipe) error {
	if recipe == nil {
		return fmt.Errorf("recipe is required")
	}
	for i := range recipe.Steps {
		scopePath := strings.TrimSpace(recipe.Steps[i].Scope.Path)
		if scopePath == "" {
			continue
		}
		absPath, err := filepath.Abs(scopePath)
		if err != nil {
			return fmt.Errorf("step %q scope path: %w", recipe.Steps[i].Name, err)
		}
		if _, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("step %q scope path not accessible: %w", recipe.Steps[i].Name, err)
		}
		recipe.Steps[i].Scope.Path = absPath
	}
	return nil
}

func formatRecipeResponse(result *core.RecipeResult) string {
	if result == nil {
		return "Recipe returned no result"
	}

	mode := ""
	modifiedLabel := "Files modified"
	if result.DryRun {
		mode = " [DRY RUN]"
		modifiedLabel = "Files that would be modified"
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, "Recipe %s completed%s\n\n", result.Name, mode)
	fmt.Fprintf(&builder, "Steps run: %d\n", result.StepsRun)
	fmt.Fprintf(&builder, "Files scanned: %d\n", result.FilesScanned)
	fmt.Fprintf(&builder, "%s: %d\n", modifiedLabel, result.FilesModified)
	fmt.Fprintf(&builder, "Total matches: %d\n", result.TotalMatches)

	if len(result.TransactionIDs) > 0 {
		builder.WriteString("\nTransactions:\n")
		for _, id := range result.TransactionIDs {
			fmt.Fprintf(&builder, "- %s\n", id)
		}
	}

	if len(result.Steps) > 0 {
		builder.WriteString("\nSteps:\n")
		for _, step := range result.Steps {
			stepResult := step.Result
			if step.AppliedResult != nil {
				stepResult = step.AppliedResult
			}
			if stepResult == nil {
				fmt.Fprintf(&builder, "- %s (%s): no result\n", step.Name, step.Method)
				continue
			}
			fmt.Fprintf(
				&builder,
				"- %s (%s): %d matches, %d files, confidence %.3f >= %.3f\n",
				step.Name,
				step.Method,
				stepResult.TotalMatches,
				stepResult.FilesModified,
				stepResult.Confidence.Score,
				step.MinConfidence,
			)
		}
	}

	if result.DryRun {
		builder.WriteString("\nThis was a dry run. No files were modified.\n")
	}

	return builder.String()
}
