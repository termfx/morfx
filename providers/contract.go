package providers

import (
	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/providers/catalog"
)

// Provider interface for language-specific implementations
type Provider interface {
	// Metadata
	Language() string
	Extensions() []string
	SupportedQueryTypes() []string

	// Core operations
	Query(source string, query core.AgentQuery) core.QueryResult
	Transform(source string, op core.TransformOp) core.TransformResult
	Validate(source string) ValidationResult

	// Observability
	Stats() Stats
}

// ValidationResult from syntax check
type ValidationResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// Registry manages all providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Language()] = provider
	catalog.Register(catalog.LanguageInfo{
		ID:         provider.Language(),
		Extensions: provider.Extensions(),
	})
}

// Get retrieves provider by language
func (r *Registry) Get(language string) (Provider, bool) {
	p, exists := r.providers[language]
	return p, exists
}

// List returns all providers
func (r *Registry) List() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// Languages returns all registered language identifiers
func (r *Registry) Languages() []string {
	langs := make([]string, 0, len(r.providers))
	for k := range r.providers {
		langs = append(langs, k)
	}
	return langs
}

// Stats captures parser-pool level metrics exposed by providers.
type Stats struct {
	BorrowCount int64 `json:"borrow_count"`
	ReturnCount int64 `json:"return_count"`
	Active      int64 `json:"active"`
}
