package providers

import (
	"github.com/termfx/morfx/core"
)

// Provider interface for language-specific implementations
type Provider interface {
	// Metadata
	Language() string
	Extensions() []string
	
	// Core operations
	Query(source string, query core.AgentQuery) core.QueryResult
	Transform(source string, op core.TransformOp) core.TransformResult
	Validate(source string) ValidationResult
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
}

// Get retrieves provider by language
func (r *Registry) Get(language string) (Provider, bool) {
	p, exists := r.providers[language]
	return p, exists
}
