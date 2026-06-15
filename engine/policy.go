package engine

import (
	"fmt"
	"path/filepath"
	"strings"
)

type rootPolicy struct {
	allowed []string
}

func newRootPolicy(roots []string) rootPolicy {
	allowed := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}

		if absRoot, err := filepath.Abs(root); err == nil {
			root = absRoot
		}

		allowed = append(allowed, filepath.Clean(root))
	}

	return rootPolicy{allowed: allowed}
}

func (p rootPolicy) ValidatePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("normalize path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	if len(p.allowed) == 0 {
		return absPath, nil
	}

	for _, root := range p.allowed {
		rel, err := filepath.Rel(root, absPath)
		if err != nil {
			continue
		}
		if rel == "." {
			return absPath, nil
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("path %q is outside allowed roots", absPath)
}
