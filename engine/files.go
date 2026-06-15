package engine

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oxhq/morfx/core"
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

func (e *Engine) FileQuery(ctx context.Context, req FileQueryRequest) (FileQueryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(req.Scope.Path) == "" {
		return FileQueryResult{}, fmt.Errorf("scope.path is required")
	}

	policy := newRootPolicy(e.cfg.AllowedRoots)
	path, err := policy.ValidatePath(req.Scope.Path)
	if err != nil {
		return FileQueryResult{}, err
	}
	req.Scope.Path = path

	query := req.Query
	if strings.TrimSpace(req.DSL) != "" {
		parsed, err := core.ParseAgentQueryPayload(nil, req.DSL)
		if err != nil {
			return FileQueryResult{}, err
		}
		query = parsed
	}

	matches, err := e.runtime.FileProcessor.QueryFiles(ctx, req.Scope, query)
	if err != nil {
		return FileQueryResult{}, err
	}

	return FileQueryResult{Results: matches}, nil
}

func (e *Engine) FileReplace(ctx context.Context, req FileReplaceRequest) (FileReplaceResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(req.Scope.Path) == "" {
		return FileReplaceResult{}, fmt.Errorf("scope.path is required")
	}

	policy := newRootPolicy(e.cfg.AllowedRoots)
	path, err := policy.ValidatePath(req.Scope.Path)
	if err != nil {
		return FileReplaceResult{}, err
	}
	req.Scope.Path = path

	result, err := e.runtime.FileProcessor.TransformFiles(ctx, core.FileTransformOp{
		TransformOp: req.Op,
		Scope:       req.Scope,
		DryRun:      req.DryRun,
		Backup:      req.Backup,
		Parallel:    true,
	})
	if err != nil {
		return FileReplaceResult{}, err
	}

	return FileReplaceResult{
		FilesScanned:  result.FilesScanned,
		FilesModified: result.FilesModified,
		TotalMatches:  result.TotalMatches,
		Errors:        result.Errors,
		TransactionID: result.TransactionID,
		Details:       result.Files,
	}, nil
}
