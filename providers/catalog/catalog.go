package catalog

import (
	"sort"
	"strings"
	"sync"
)

// LanguageInfo captures metadata about a language provider.
type LanguageInfo struct {
	ID         string
	Extensions []string
}

var (
	mu     sync.RWMutex
	byLang = make(map[string]LanguageInfo)
	byExt  = make(map[string]LanguageInfo)
)

// Register stores language metadata for extension lookups. Subsequent
// registrations for the same language overwrite prior data to keep the catalog
// in sync with the latest provider definition.
func Register(info LanguageInfo) {
	if info.ID == "" {
		return
	}

	normalized := uniqueExtensions(info.Extensions)
	info.Extensions = normalized

	mu.Lock()
	defer mu.Unlock()

	byLang[strings.ToLower(info.ID)] = info
	for _, ext := range normalized {
		byExt[ext] = info
	}
}

// LookupByExtension returns the language info associated with a file extension.
func LookupByExtension(ext string) (LanguageInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	info, ok := byExt[strings.ToLower(ext)]
	return info, ok
}

// Languages returns all registered language infos sorted by language ID.
func Languages() []LanguageInfo {
	mu.RLock()
	defer mu.RUnlock()

	infos := make([]LanguageInfo, 0, len(byLang))
	for _, info := range byLang {
		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}

func uniqueExtensions(exts []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(exts))
	for _, ext := range exts {
		normalized := strings.ToLower(strings.TrimSpace(ext))
		if normalized == "" {
			continue
		}
		if !strings.HasPrefix(normalized, ".") {
			normalized = "." + normalized
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
