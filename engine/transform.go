package engine

import (
	"context"
	"fmt"
	"strings"
)

func (e *Engine) Transform(_ context.Context, req TransformRequest) (TransformResult, error) {
	if strings.TrimSpace(req.Language) == "" {
		return TransformResult{}, fmt.Errorf("language is required")
	}

	provider, ok := e.runtime.Providers.Get(req.Language)
	if !ok {
		return TransformResult{}, fmt.Errorf("provider not found: %s", req.Language)
	}

	result := provider.Transform(req.Source, req.Op)
	if result.Error != nil {
		return TransformResult{}, result.Error
	}

	return TransformResult{
		MatchCount: result.MatchCount,
		Modified:   result.Modified,
		Diff:       result.Diff,
		Confidence: result.Confidence,
	}, nil
}
