package prompts

import (
	"testing"

	"github.com/termfx/morfx/mcp/types"
)

func TestNewCodeReviewPrompt(t *testing.T) {
	prompt := NewCodeReviewPrompt()

	if prompt.Name() != "code_review" {
		t.Errorf("Expected name 'code_review', got '%s'", prompt.Name())
	}

	if prompt.Description() == "" {
		t.Error("Description should not be empty")
	}

	if prompt.Content() == "" {
		t.Error("Content should not be empty")
	}

	args := prompt.Arguments()
	if len(args) != 3 {
		t.Errorf("Expected 3 arguments, got %d", len(args))
	}

	// Check required arguments
	if args[0].Name != "code" || !args[0].Required {
		t.Error("First argument should be 'code' and required")
	}

	if args[1].Name != "language" || !args[1].Required {
		t.Error("Second argument should be 'language' and required")
	}

	if args[2].Name != "focus_areas" || args[2].Required {
		t.Error("Third argument should be 'focus_areas' and optional")
	}
}

func TestNewRefactorPrompt(t *testing.T) {
	prompt := NewRefactorPrompt()

	if prompt.Name() != "refactor" {
		t.Errorf("Expected name 'refactor', got '%s'", prompt.Name())
	}

	args := prompt.Arguments()
	if len(args) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(args))
	}

	if args[0].Name != "code" || !args[0].Required {
		t.Error("First argument should be 'code' and required")
	}
}

func TestNewTestGenerationPrompt(t *testing.T) {
	prompt := NewTestGenerationPrompt()

	if prompt.Name() != "test_generation" {
		t.Errorf("Expected name 'test_generation', got '%s'", prompt.Name())
	}

	args := prompt.Arguments()
	if len(args) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(args))
	}
}

func TestNewDocumentationPrompt(t *testing.T) {
	prompt := NewDocumentationPrompt()

	if prompt.Name() != "documentation" {
		t.Errorf("Expected name 'documentation', got '%s'", prompt.Name())
	}

	args := prompt.Arguments()
	if len(args) != 2 {
		t.Errorf("Expected 2 arguments, got %d", len(args))
	}
}

func TestBasePromptInterface(t *testing.T) {
	prompt := &BasePrompt{
		name:        "test",
		description: "Test prompt",
		content:     "Test content",
		arguments: []types.PromptArgument{
			{Name: "arg1", Description: "First arg", Required: true},
		},
	}

	// Test interface compliance
	var _ types.Prompt = prompt

	if prompt.Name() != "test" {
		t.Errorf("Expected name 'test', got '%s'", prompt.Name())
	}

	if prompt.Description() != "Test prompt" {
		t.Errorf("Expected description 'Test prompt', got '%s'", prompt.Description())
	}

	if prompt.Content() != "Test content" {
		t.Errorf("Expected content 'Test content', got '%s'", prompt.Content())
	}

	args := prompt.Arguments()
	if len(args) != 1 || args[0].Name != "arg1" {
		t.Error("Arguments not returned correctly")
	}
}

func TestPromptRegistry_Register(t *testing.T) {
	registry := &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}

	prompt := &BasePrompt{
		name:        "test_prompt",
		description: "Test prompt",
		content:     "Test content",
	}

	registry.Register("test", prompt)

	if len(registry.prompts) != 1 {
		t.Errorf("Expected 1 prompt, got %d", len(registry.prompts))
	}

	if len(registry.ordered) != 1 {
		t.Errorf("Expected 1 ordered prompt, got %d", len(registry.ordered))
	}

	if registry.ordered[0] != "test" {
		t.Errorf("Expected ordered prompt 'test', got '%s'", registry.ordered[0])
	}
}

func TestPromptRegistry_Get(t *testing.T) {
	registry := &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}

	prompt := &BasePrompt{name: "test_prompt"}
	registry.Register("test", prompt)

	// Test existing prompt
	retrieved, exists := registry.Get("test")
	if !exists {
		t.Error("Prompt should exist")
	}

	if retrieved.Name() != "test_prompt" {
		t.Errorf("Expected name 'test_prompt', got '%s'", retrieved.Name())
	}

	// Test non-existing prompt
	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Non-existent prompt should not exist")
	}
}

func TestPromptRegistry_List(t *testing.T) {
	registry := &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}

	prompt1 := &BasePrompt{name: "prompt1"}
	prompt2 := &BasePrompt{name: "prompt2"}

	registry.Register("first", prompt1)
	registry.Register("second", prompt2)

	prompts := registry.List()
	if len(prompts) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(prompts))
	}

	// Check order is preserved
	if prompts[0].Name() != "prompt1" {
		t.Errorf("Expected first prompt 'prompt1', got '%s'", prompts[0].Name())
	}

	if prompts[1].Name() != "prompt2" {
		t.Errorf("Expected second prompt 'prompt2', got '%s'", prompts[1].Name())
	}
}

func TestRegisterAll(t *testing.T) {
	// Clear registry for clean test
	Registry = &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}

	RegisterAll()

	prompts := Registry.List()
	if len(prompts) != 4 {
		t.Errorf("Expected 4 registered prompts, got %d", len(prompts))
	}

	expectedPrompts := []string{"code_review", "refactor", "test_generation", "documentation"}
	for _, expected := range expectedPrompts {
		_, exists := Registry.Get(expected)
		if !exists {
			t.Errorf("Expected prompt '%s' to be registered", expected)
		}
	}
}

func TestGet(t *testing.T) {
	// Clear and initialize registry
	Registry = &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}
	RegisterAll()

	prompt, exists := Get("code_review")
	if !exists {
		t.Error("code_review prompt should exist")
	}

	if prompt.Name() != "code_review" {
		t.Errorf("Expected name 'code_review', got '%s'", prompt.Name())
	}
}

func TestGetDefinitions(t *testing.T) {
	// Clear and initialize registry
	Registry = &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}
	RegisterAll()

	definitions := GetDefinitions()
	if len(definitions) != 4 {
		t.Errorf("Expected 4 definitions, got %d", len(definitions))
	}

	// Check first definition structure
	def := definitions[0]
	if def.Name == "" {
		t.Error("Definition name should not be empty")
	}

	if def.Description == "" {
		t.Error("Definition description should not be empty")
	}

	if len(def.Arguments) == 0 {
		t.Error("Definition should have arguments")
	}
}

func TestInit(t *testing.T) {
	// Clear registry
	Registry = &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}

	Init()

	prompts := Registry.List()
	if len(prompts) == 0 {
		t.Error("Init should register prompts")
	}
}

func TestPromptRegistry_DuplicateRegistration(t *testing.T) {
	registry := &promptRegistry{
		prompts: make(map[string]types.Prompt),
		ordered: make([]string, 0),
	}

	prompt1 := &BasePrompt{name: "first"}
	prompt2 := &BasePrompt{name: "second"}

	registry.Register("test", prompt1)
	registry.Register("test", prompt2) // Duplicate key

	// Should have only one entry in ordered list
	if len(registry.ordered) != 1 {
		t.Errorf("Expected 1 ordered entry, got %d", len(registry.ordered))
	}

	// Should overwrite the prompt
	retrieved, _ := registry.Get("test")
	if retrieved.Name() != "second" {
		t.Errorf("Expected overwritten prompt 'second', got '%s'", retrieved.Name())
	}
}
