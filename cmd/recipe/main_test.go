package main

import (
	"context"
	"strings"
	"testing"

	"github.com/oxhq/morfx/core"
)

type recipeEngineSpy struct {
	called bool
	recipe core.Recipe
	result *core.RecipeResult
	err    error
}

func (s *recipeEngineSpy) Recipe(_ context.Context, recipe core.Recipe) (*core.RecipeResult, error) {
	s.called = true
	s.recipe = recipe
	return s.result, s.err
}

func TestRunRecipeRoutesToEngine(t *testing.T) {
	root := t.TempDir()
	spy := &recipeEngineSpy{
		result: &core.RecipeResult{
			Name:     "rename-handlers",
			DryRun:   true,
			StepsRun: 1,
		},
	}

	req := core.Recipe{
		Name:   "rename-handlers",
		DryRun: true,
		Steps: []core.RecipeStep{{
			Name:   "rename handler",
			Method: "replace",
			Scope: core.FileScope{
				Path: root,
			},
			TargetDSL:   "func:Hello",
			Replacement: "func Renamed() {}",
		}},
	}

	payload, err := runRecipe(context.Background(), spy, &req)
	if err != nil {
		t.Fatalf("runRecipe() error = %v", err)
	}
	if !spy.called {
		t.Fatal("expected runRecipe() to call engine")
	}
	if spy.recipe.Name != "rename-handlers" {
		t.Fatalf("engine recipe name = %q, want %q", spy.recipe.Name, "rename-handlers")
	}
	if spy.recipe.Steps[0].Scope.Path != root {
		t.Fatalf("engine scope path = %q, want %q", spy.recipe.Steps[0].Scope.Path, root)
	}
	if got := payload["name"]; got != "rename-handlers" {
		t.Fatalf("payload name = %v, want %q", got, "rename-handlers")
	}
}

func TestFormatRecipeResponseIncludesAggregateCounts(t *testing.T) {
	result := &core.RecipeResult{
		Name:          "rename-handlers",
		DryRun:        true,
		StepsRun:      1,
		FilesScanned:  2,
		FilesModified: 1,
		TotalMatches:  3,
		Steps: []core.RecipeStepResult{{
			Name:          "rename handler",
			Method:        "replace",
			DryRun:        true,
			MinConfidence: 0.85,
			Result: &core.FileTransformResult{
				FilesScanned:  2,
				FilesModified: 1,
				TotalMatches:  3,
				Confidence:    core.ConfidenceScore{Score: 0.92, Level: "high"},
			},
		}},
	}

	text := formatRecipeResponse(result)

	for _, want := range []string{
		"Recipe rename-handlers completed [DRY RUN]",
		"Steps run: 1",
		"Files scanned: 2",
		"Files that would be modified: 1",
		"Total matches: 3",
		"rename handler",
		"confidence 0.920",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected response to contain %q, got:\n%s", want, text)
		}
	}
}
