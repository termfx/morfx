package mcp

import (
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
)

type paginationParams struct {
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func applyPagination[T any](items []T, cursor string, limit int) ([]T, *string, error) {
	if limit <= 0 {
		if len(items) < defaultListLimit {
			limit = len(items)
		} else {
			limit = defaultListLimit
		}
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	start := 0
	if cursor != "" {
		idx, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cursor value: %w", err)
		}
		if idx < 0 || idx > len(items) {
			return nil, nil, fmt.Errorf("cursor out of range")
		}
		start = idx
	}

	if start >= len(items) {
		return []T{}, nil, nil
	}

	end := start + limit
	if end > len(items) {
		end = len(items)
	}

	page := items[start:end]

	if end >= len(items) {
		return page, nil, nil
	}

	next := strconv.Itoa(end)
	return page, &next, nil
}

func decodePaginationParams(raw json.RawMessage) (paginationParams, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return paginationParams{}, nil
	}
	var params paginationParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return paginationParams{}, err
	}
	return params, nil
}
