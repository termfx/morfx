package prompts

import (
	"sync"

	"github.com/termfx/morfx/mcp/types"
)

// BasePrompt provides common prompt functionality
type BasePrompt struct {
	name        string
	description string
	content     string
	arguments   []types.PromptArgument
}

// Name returns the prompt name
func (p *BasePrompt) Name() string {
	return p.name
}

// Description returns the prompt description
func (p *BasePrompt) Description() string {
	return p.description
}

// Content returns the prompt content
func (p *BasePrompt) Content() string {
	return p.content
}

// Arguments returns the prompt arguments
func (p *BasePrompt) Arguments() []types.PromptArgument {
	return p.arguments
}

// promptRegistry manages prompts internally
type promptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]types.Prompt
	ordered []string
}

// Registry holds all registered prompts
var Registry = &promptRegistry{
	prompts: make(map[string]types.Prompt),
	ordered: make([]string, 0),
}

// Register adds a prompt to the registry
func (r *promptRegistry) Register(name string, prompt types.Prompt) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.prompts[name]; !exists {
		r.ordered = append(r.ordered, name)
	}
	r.prompts[name] = prompt
}

// Get retrieves a prompt by name
func (r *promptRegistry) Get(name string) (types.Prompt, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	prompt, exists := r.prompts[name]
	return prompt, exists
}

// List returns all prompts in registration order
func (r *promptRegistry) List() []types.Prompt {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.Prompt, 0, len(r.ordered))
	for _, name := range r.ordered {
		result = append(result, r.prompts[name])
	}
	return result
}

// Init initializes the prompt registry
func Init() {
	RegisterAll()
}

// RegisterAll registers all built-in prompts
func RegisterAll() {
	// Register all prompts here
	Registry.Register("code_review", NewCodeReviewPrompt())
	Registry.Register("refactor", NewRefactorPrompt())
	Registry.Register("test_generation", NewTestGenerationPrompt())
	Registry.Register("documentation", NewDocumentationPrompt())
}

// Get retrieves a prompt by name
func Get(name string) (types.Prompt, bool) {
	return Registry.Get(name)
}

// GetDefinitions returns all prompt definitions
func GetDefinitions() []types.PromptDefinition {
	prompts := Registry.List()
	definitions := make([]types.PromptDefinition, 0, len(prompts))

	for _, prompt := range prompts {
		definitions = append(definitions, types.PromptDefinition{
			Name:        prompt.Name(),
			Description: prompt.Description(),
			Arguments:   prompt.Arguments(),
		})
	}

	return definitions
}
