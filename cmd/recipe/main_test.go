package main

import (
	"strings"
	"testing"

	"github.com/oxhq/morfx/core"
)

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
