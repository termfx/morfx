package core

import (
	"context"
	"fmt"
	"strings"
)

const DefaultRecipeMinConfidence = 0.85

// Recipe is a named, repeatable transformation made from Morfx primitives.
type Recipe struct {
	Name          string       `json:"name"`
	Description   string       `json:"description,omitempty"`
	DryRun        bool         `json:"dry_run,omitempty"`
	MinConfidence float64      `json:"min_confidence,omitempty"`
	Steps         []RecipeStep `json:"steps"`
}

// RecipeStep defines one query plus transform operation over a file scope.
type RecipeStep struct {
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Method        string     `json:"method"`
	Scope         FileScope  `json:"scope"`
	Target        AgentQuery `json:"target"`
	Replacement   string     `json:"replacement,omitempty"`
	Content       string     `json:"content,omitempty"`
	MinConfidence float64    `json:"min_confidence,omitempty"`
	Backup        bool       `json:"backup,omitempty"`
}

// Rule is an alias for one reusable recipe step.
type Rule = RecipeStep

// RecipeResult is the aggregate execution result for a recipe.
type RecipeResult struct {
	Name           string             `json:"name"`
	DryRun         bool               `json:"dry_run"`
	StepsRun       int                `json:"steps_run"`
	FilesScanned   int                `json:"files_scanned"`
	FilesModified  int                `json:"files_modified"`
	TotalMatches   int                `json:"total_matches"`
	TransactionIDs []string           `json:"transaction_ids,omitempty"`
	Steps          []RecipeStepResult `json:"steps"`
}

// RecipeStepResult records the preflight/apply outcome for one recipe step.
type RecipeStepResult struct {
	Name          string               `json:"name"`
	Method        string               `json:"method"`
	DryRun        bool                 `json:"dry_run"`
	MinConfidence float64              `json:"min_confidence"`
	Result        *FileTransformResult `json:"result"`
	AppliedResult *FileTransformResult `json:"applied_result,omitempty"`
}

// RecipeProcessor is the file transformation boundary used by recipes.
type RecipeProcessor interface {
	TransformFiles(context.Context, FileTransformOp) (*FileTransformResult, error)
}

// ValidateRecipe checks a recipe before execution.
func ValidateRecipe(recipe Recipe) error {
	if strings.TrimSpace(recipe.Name) == "" {
		return fmt.Errorf("recipe name is required")
	}
	if len(recipe.Steps) == 0 {
		return fmt.Errorf("recipe must contain at least one step")
	}
	for i, step := range recipe.Steps {
		if err := validateRecipeStep(i, step); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteRecipe preflights every apply-mode step before mutating files.
func ExecuteRecipe(ctx context.Context, processor RecipeProcessor, recipe Recipe) (*RecipeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if processor == nil {
		return nil, fmt.Errorf("recipe processor is required")
	}
	if err := ValidateRecipe(recipe); err != nil {
		return nil, err
	}

	result := &RecipeResult{
		Name:           recipe.Name,
		DryRun:         recipe.DryRun,
		TransactionIDs: make([]string, 0),
		Steps:          make([]RecipeStepResult, 0, len(recipe.Steps)),
	}

	for _, step := range recipe.Steps {
		stepThreshold := recipeStepThreshold(recipe, step)
		preflightOp := recipeStepOperation(step, true)
		preflight, err := processor.TransformFiles(ctx, preflightOp)
		if err != nil {
			return nil, fmt.Errorf("step %q preflight failed: %w", step.Name, err)
		}
		if err := requireRecipeConfidence(step.Name, preflight, stepThreshold); err != nil {
			return nil, err
		}

		stepResult := RecipeStepResult{
			Name:          step.Name,
			Method:        step.Method,
			DryRun:        true,
			MinConfidence: stepThreshold,
			Result:        preflight,
		}

		finalResult := preflight
		if !recipe.DryRun {
			applyOp := recipeStepOperation(step, false)
			applied, err := processor.TransformFiles(ctx, applyOp)
			if err != nil {
				return nil, fmt.Errorf("step %q apply failed: %w", step.Name, err)
			}
			stepResult.DryRun = false
			stepResult.AppliedResult = applied
			finalResult = applied
		}

		result.StepsRun++
		result.FilesScanned += finalResult.FilesScanned
		result.FilesModified += finalResult.FilesModified
		result.TotalMatches += finalResult.TotalMatches
		if finalResult.TransactionID != "" {
			result.TransactionIDs = append(result.TransactionIDs, finalResult.TransactionID)
		}
		result.Steps = append(result.Steps, stepResult)
	}

	return result, nil
}

func validateRecipeStep(index int, step RecipeStep) error {
	prefix := fmt.Sprintf("step %d", index+1)
	if strings.TrimSpace(step.Name) == "" {
		return fmt.Errorf("%s name is required", prefix)
	}
	if !isSupportedRecipeMethod(step.Method) {
		return fmt.Errorf("%s unsupported method: %s", prefix, step.Method)
	}
	if strings.TrimSpace(step.Scope.Path) == "" {
		return fmt.Errorf("%s scope.path is required", prefix)
	}
	if step.Target.Type == "" && step.Method != "append" {
		return fmt.Errorf("%s target.type is required", prefix)
	}
	switch step.Method {
	case "replace":
		if strings.TrimSpace(step.Replacement) == "" {
			return fmt.Errorf("%s replacement is required", prefix)
		}
	case "insert_before", "insert_after", "append":
		if strings.TrimSpace(step.Content) == "" {
			return fmt.Errorf("%s content is required", prefix)
		}
	}
	if step.MinConfidence < 0 || step.MinConfidence > 1 {
		return fmt.Errorf("%s min_confidence must be between 0 and 1", prefix)
	}
	return nil
}

func isSupportedRecipeMethod(method string) bool {
	switch method {
	case "replace", "delete", "insert_before", "insert_after", "append":
		return true
	default:
		return false
	}
}

func recipeStepOperation(step RecipeStep, dryRun bool) FileTransformOp {
	return FileTransformOp{
		TransformOp: TransformOp{
			Method:      step.Method,
			Target:      step.Target,
			Content:     step.Content,
			Replacement: step.Replacement,
		},
		Scope:    step.Scope,
		DryRun:   dryRun,
		Backup:   step.Backup,
		Parallel: true,
	}
}

func recipeStepThreshold(recipe Recipe, step RecipeStep) float64 {
	if step.MinConfidence > 0 {
		return step.MinConfidence
	}
	if recipe.MinConfidence > 0 {
		return recipe.MinConfidence
	}
	return DefaultRecipeMinConfidence
}

func requireRecipeConfidence(stepName string, result *FileTransformResult, threshold float64) error {
	if result == nil {
		return fmt.Errorf("step %q returned no result", stepName)
	}
	if result.FilesModified == 0 && result.TotalMatches == 0 {
		return nil
	}
	if result.Confidence.Score < threshold {
		return fmt.Errorf(
			"step %q confidence %.3f is below required %.3f",
			stepName,
			result.Confidence.Score,
			threshold,
		)
	}
	return nil
}
