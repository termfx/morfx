package lang

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/garaekz/fileman/internal/lang/golang"
	"github.com/garaekz/fileman/internal/provider"
)

// registry contains all available language providers.
type registry struct {
	providers map[string]provider.LanguageProvider
}

// defaultRegistry is a singleton instance that is automatically initialized.
var defaultRegistry *registry

func init() {
	defaultRegistry = &registry{
		providers: make(map[string]provider.LanguageProvider),
	}

	register(golang.New())
	// register(python.New())
}

// register adds a provider to the registry using all its aliases.
func register(p provider.LanguageProvider) {
	for _, alias := range p.Aliases() {
		defaultRegistry.providers[alias] = p
	}
}

// ResolveProvider determines the correct language provider.
// It prioritizes the explicit user flag and then tries to infer from the files.
func ResolveProvider(langFlag string, files []string) (provider.LanguageProvider, error) {
	// Priority 1: Use the language flag if provided.
	if langFlag != "" {
		p, ok := defaultRegistry.providers[langFlag]
		if !ok {
			return nil, fmt.Errorf("unsupported language specified: %s", langFlag)
		}
		return p, nil
	}

	// Priority 2: Infer the language from file extensions.

	// Because single entry point of this cli we know that files are provided.
	// And if only one file it does have an extension.
	if len(files) == 1 {
		ext := strings.TrimPrefix(filepath.Ext(files[0]), ".")
		if ext != "" {
			p, ok := defaultRegistry.providers[ext]
			if ok {
				return p, nil
			}
		}
		return nil, errors.New("could not infer language from file extension, please specify with --lang")
	}

	return resolveProviderFromFilePaths(files)
}

// resolveProviderFromFilePaths tries to deduce the dominant language from a list of files.
func resolveProviderFromFilePaths(files []string) (provider.LanguageProvider, error) {
	differentLangsCount := 0
	counts := make(map[string]int)
	for _, file := range files {
		ext := strings.TrimPrefix(filepath.Ext(file), ".")
		if ext != "" {
			if _, ok := counts[ext]; !ok {
				differentLangsCount++
			}
			counts[ext]++
		}
	}

	if differentLangsCount == 1 {
		// Only one language detected try to resolve it.
		for ext := range counts {
			return defaultRegistry.providers[ext], nil
		}

		return nil, fmt.Errorf("we could not infer the language from filepath: %s", files[0])
	}

	var maxCount int
	var dominantLang string
	for ext, count := range counts {
		if count > maxCount {
			maxCount = count
			dominantLang = ext
		}
	}
	if dominantLang != "" {
		return defaultRegistry.providers[dominantLang], nil
	}

	return nil, fmt.Errorf("we could not infer the language from file paths: %v", files)
}
