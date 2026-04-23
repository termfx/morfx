package toolenv

import (
	"fmt"
	"os"
	"strings"
)

// SourceData encapsulates the resolved source code for single-file tools.
type SourceData struct {
	Code     string
	Path     string
	Perm     os.FileMode
	FromFile bool
}

// LoadSource resolves code from either an inline source field or a filesystem path.
// Exactly one input must be provided. The function preserves file permissions when reading
// from disk so callers can write modifications back safely.
func LoadSource(source *string, path *string) (*SourceData, error) {
	sourceProvided := source != nil && path == nil
	pathProvided := path != nil && source == nil && strings.TrimSpace(*path) != ""

	if sourceProvided == pathProvided {
		return nil, fmt.Errorf("exactly one of 'source' or 'path' must be provided")
	}

	if pathProvided {
		content, err := os.ReadFile(*path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", *path, err)
		}
		perm := os.FileMode(0o644)
		if info, err := os.Stat(*path); err == nil {
			perm = info.Mode().Perm()
		}
		return &SourceData{
			Code:     string(content),
			Path:     *path,
			Perm:     perm,
			FromFile: true,
		}, nil
	}

	// Inline source mode (can be empty string legitimately)
	if source == nil {
		return nil, fmt.Errorf("source content missing")
	}

	return &SourceData{
		Code:     *source,
		FromFile: false,
	}, nil
}
