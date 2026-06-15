package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/oxhq/morfx/core"
)

func (e *Engine) Query(_ context.Context, req QueryRequest) (QueryResult, error) {
	if strings.TrimSpace(req.Language) == "" {
		return QueryResult{}, fmt.Errorf("language is required")
	}

	provider, ok := e.runtime.Providers.Get(req.Language)
	if !ok {
		return QueryResult{}, fmt.Errorf("provider not found: %s", req.Language)
	}

	query := req.Query
	if strings.TrimSpace(req.DSL) != "" {
		parsed, err := core.ParseAgentQueryPayload(nil, req.DSL)
		if err != nil {
			return QueryResult{}, err
		}
		query = parsed
	}

	result := provider.Query(req.Source, query)
	if result.Error != nil {
		return QueryResult{}, result.Error
	}

	return QueryResult{
		Matches: result.Total,
		Results: result.Matches,
	}, nil
}
