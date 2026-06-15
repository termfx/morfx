package engine

import (
	"context"
	"strings"

	"github.com/oxhq/morfx/core"
)

func (e *Engine) Recipe(ctx context.Context, recipe core.Recipe) (*core.RecipeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	validated := recipe
	policy := newRootPolicy(e.cfg.AllowedRoots)
	for i := range validated.Steps {
		scopePath := strings.TrimSpace(validated.Steps[i].Scope.Path)
		if scopePath == "" {
			continue
		}
		path, err := policy.ValidatePath(scopePath)
		if err != nil {
			return nil, err
		}
		validated.Steps[i].Scope.Path = path
	}

	return core.ExecuteRecipe(ctx, e.runtime.FileProcessor, validated)
}
