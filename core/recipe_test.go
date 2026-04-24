package core

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRecipeProcessor struct {
	results []*FileTransformResult
	calls   []FileTransformOp
}

func (p *fakeRecipeProcessor) TransformFiles(_ context.Context, op FileTransformOp) (*FileTransformResult, error) {
	p.calls = append(p.calls, op)
	if len(p.results) == 0 {
		return nil, errors.New("unexpected transform call")
	}
	result := p.results[0]
	p.results = p.results[1:]
	return result, nil
}

func TestExecuteRecipeDryRunRunsEachStepWithoutApply(t *testing.T) {
	processor := &fakeRecipeProcessor{
		results: []*FileTransformResult{
			{
				FilesScanned:  2,
				FilesModified: 1,
				TotalMatches:  1,
				Confidence:    ConfidenceScore{Score: 0.96, Level: "high"},
			},
		},
	}

	recipe := Recipe{
		Name:   "rename-handlers",
		DryRun: true,
		Steps: []RecipeStep{{
			Name:        "rename handler",
			Method:      "replace",
			Scope:       FileScope{Path: ".", Include: []string{"**/*.go"}, Language: "go"},
			Target:      AgentQuery{Type: "function", Name: "OldHandler"},
			Replacement: "func NewHandler() {}",
		}},
	}

	result, err := ExecuteRecipe(context.Background(), processor, recipe)
	if err != nil {
		t.Fatalf("ExecuteRecipe returned error: %v", err)
	}

	if !result.DryRun {
		t.Fatal("recipe result should be marked as dry run")
	}
	if result.StepsRun != 1 || result.FilesModified != 1 || result.TotalMatches != 1 {
		t.Fatalf("unexpected aggregate result: %+v", result)
	}
	if len(processor.calls) != 1 {
		t.Fatalf("expected one processor call, got %d", len(processor.calls))
	}
	if !processor.calls[0].DryRun {
		t.Fatal("dry-run recipe must not apply the transform")
	}
	if processor.calls[0].Method != "replace" || processor.calls[0].Target.Name != "OldHandler" {
		t.Fatalf("unexpected transform op: %+v", processor.calls[0].TransformOp)
	}
}

func TestExecuteRecipeApplyPreflightsConfidenceBeforeMutating(t *testing.T) {
	processor := &fakeRecipeProcessor{
		results: []*FileTransformResult{
			{
				FilesScanned:  1,
				FilesModified: 1,
				TotalMatches:  1,
				Confidence:    ConfidenceScore{Score: 0.91, Level: "high"},
			},
			{
				FilesScanned:      1,
				FilesModified:     1,
				TotalMatches:      1,
				Confidence:        ConfidenceScore{Score: 0.91, Level: "high"},
				TransactionID:     "tx-123",
				TransformDuration: 3,
			},
		},
	}

	recipe := Recipe{
		Name:          "remove-debug-functions",
		MinConfidence: 0.9,
		Steps: []RecipeStep{{
			Name:   "delete debug helpers",
			Method: "delete",
			Scope:  FileScope{Path: ".", Include: []string{"**/*.go"}, Language: "go"},
			Target: AgentQuery{Type: "function", Name: "Debug*"},
		}},
	}

	result, err := ExecuteRecipe(context.Background(), processor, recipe)
	if err != nil {
		t.Fatalf("ExecuteRecipe returned error: %v", err)
	}

	if result.DryRun {
		t.Fatal("apply recipe should not be marked as dry run")
	}
	if len(processor.calls) != 2 {
		t.Fatalf("expected preflight and apply calls, got %d", len(processor.calls))
	}
	if !processor.calls[0].DryRun {
		t.Fatal("first apply-mode call must be dry-run preflight")
	}
	if processor.calls[1].DryRun {
		t.Fatal("second apply-mode call must apply the transform")
	}
	if result.TransactionIDs[0] != "tx-123" {
		t.Fatalf("expected transaction id to be preserved, got %+v", result.TransactionIDs)
	}
}

func TestExecuteRecipeBlocksLowConfidenceBeforeApply(t *testing.T) {
	processor := &fakeRecipeProcessor{
		results: []*FileTransformResult{
			{
				FilesScanned:  3,
				FilesModified: 2,
				TotalMatches:  4,
				Confidence:    ConfidenceScore{Score: 0.62, Level: "medium"},
			},
		},
	}

	recipe := Recipe{
		Name:          "broad-delete",
		MinConfidence: 0.85,
		Steps: []RecipeStep{{
			Name:   "delete broad match",
			Method: "delete",
			Scope:  FileScope{Path: ".", Include: []string{"**/*.go"}, Language: "go"},
			Target: AgentQuery{Type: "function", Name: "*"},
		}},
	}

	_, err := ExecuteRecipe(context.Background(), processor, recipe)
	if err == nil {
		t.Fatal("expected low confidence recipe to fail")
	}
	if !strings.Contains(err.Error(), "confidence") {
		t.Fatalf("expected confidence error, got %v", err)
	}
	if len(processor.calls) != 1 || !processor.calls[0].DryRun {
		t.Fatalf("expected only dry-run preflight before failure, calls=%+v", processor.calls)
	}
}

func TestValidateRecipeRejectsInvalidSteps(t *testing.T) {
	err := ValidateRecipe(Recipe{
		Name: "invalid",
		Steps: []RecipeStep{{
			Name:   "invalid method",
			Method: "rewrite",
			Scope:  FileScope{Path: ".", Language: "go"},
			Target: AgentQuery{Type: "function", Name: "Old"},
		}},
	})
	if err == nil {
		t.Fatal("expected invalid recipe to fail validation")
	}
	if !strings.Contains(err.Error(), "unsupported method") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestRecipeStepAcceptsTargetDSL(t *testing.T) {
	processor := &fakeRecipeProcessor{
		results: []*FileTransformResult{{
			FilesScanned:  1,
			FilesModified: 1,
			TotalMatches:  1,
			Confidence:    ConfidenceScore{Score: 0.93, Level: "high"},
		}},
	}

	recipe := Recipe{
		Name:   "remove-functions-that-read-env",
		DryRun: true,
		Steps: []RecipeStep{{
			Name:      "delete env readers",
			Method:    "delete",
			Scope:     FileScope{Path: ".", Include: []string{"**/*.go"}, Language: "go"},
			TargetDSL: "func:* > call:os.Getenv",
		}},
	}

	if err := ValidateRecipe(recipe); err != nil {
		t.Fatalf("ValidateRecipe returned error: %v", err)
	}
	if _, err := ExecuteRecipe(context.Background(), processor, recipe); err != nil {
		t.Fatalf("ExecuteRecipe returned error: %v", err)
	}
	if len(processor.calls) != 1 {
		t.Fatalf("expected one processor call, got %d", len(processor.calls))
	}
	call := processor.calls[0]
	if call.Target.Type != "func" || call.Target.Contains == nil {
		t.Fatalf("expected parsed structural target, got %+v", call.Target)
	}
}
