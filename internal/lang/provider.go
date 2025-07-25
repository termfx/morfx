package lang

import "fmt"

// LanguageProvider defines the contract for language-specific query providers.
// Each provider is responsible for supplying the correct Tree-sitter query
// template for a given semantic node type.
type LanguageProvider interface {
	GetQuery(nodeType, nodeName string) (string, bool)
}

var providers = make(map[string]LanguageProvider)

// Register makes a LanguageProvider available by name.
// This function should be called from the init function of each
// language-specific package.
func Register(name string, provider LanguageProvider) {
	if provider == nil {
		panic("Register: provider is nil")
	}
	if _, dup := providers[name]; dup {
		panic("Register: called twice for provider " + name)
	}
	providers[name] = provider
}

// GetQueryForLanguage retrieves a formatted query for a given language.
// It looks up the registered provider for the specified language and uses it
// to generate the query.
func GetQueryForLanguage(lang, nodeType, nodeName string) (string, error) {
	provider, ok := providers[lang]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
	query, ok := provider.GetQuery(nodeType, nodeName)
	if !ok {
		return "", fmt.Errorf("unsupported node type '%s' for language '%s'", nodeType, lang)
	}
	return query, nil
}
