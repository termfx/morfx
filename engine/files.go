package engine

import (
	"context"
	"os"
)

func (e *Engine) FileTransform(ctx context.Context, req FileTransformRequest) (FileTransformResult, error) {
	policy := newRootPolicy(e.cfg.AllowedRoots)
	path, err := policy.ValidatePath(req.Path)
	if err != nil {
		return FileTransformResult{}, err
	}

	original, err := os.ReadFile(path)
	if err != nil {
		return FileTransformResult{}, err
	}

	transformResult, err := e.Transform(ctx, TransformRequest{
		Language: req.Language,
		Source:   string(original),
		Op:       req.Op,
	})
	if err != nil {
		return FileTransformResult{}, err
	}

	applied, err := e.writePipelineApply(path, string(original), transformResult.Modified)
	if err != nil {
		return FileTransformResult{}, err
	}

	return FileTransformResult{
		Applied:    applied,
		MatchCount: transformResult.MatchCount,
		Diff:       transformResult.Diff,
		Confidence: transformResult.Confidence,
	}, nil
}
