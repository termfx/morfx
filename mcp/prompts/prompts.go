package prompts

import "github.com/termfx/morfx/mcp/types"

// NewCodeReviewPrompt creates a code review prompt
func NewCodeReviewPrompt() *BasePrompt {
	return &BasePrompt{
		name:        "code_review",
		description: "Comprehensive code review with focus on quality and best practices",
		content: `You are an expert code reviewer. Analyze the provided code for:
1. Code quality and maintainability
2. Performance issues and optimizations
3. Security vulnerabilities
4. Adherence to language best practices
5. Test coverage and quality

Provide specific, actionable feedback with code examples where appropriate.`,
		arguments: []types.PromptArgument{
			{
				Name:        "code",
				Description: "The code to review",
				Required:    true,
			},
			{
				Name:        "language",
				Description: "Programming language of the code",
				Required:    true,
			},
			{
				Name:        "focus_areas",
				Description: "Specific areas to focus on (optional)",
				Required:    false,
			},
		},
	}
}

// NewRefactorPrompt creates a refactoring prompt
func NewRefactorPrompt() *BasePrompt {
	return &BasePrompt{
		name:        "refactor",
		description: "Suggest refactoring improvements for better code structure",
		content: `You are a refactoring expert. Analyze the code and suggest improvements for:
1. Better separation of concerns
2. Reduced complexity
3. Improved testability
4. Design pattern applications
5. Code reusability

Provide before/after examples and explain the benefits of each refactoring.`,
		arguments: []types.PromptArgument{
			{
				Name:        "code",
				Description: "The code to refactor",
				Required:    true,
			},
			{
				Name:        "goals",
				Description: "Specific refactoring goals",
				Required:    false,
			},
		},
	}
}

// NewTestGenerationPrompt creates a test generation prompt
func NewTestGenerationPrompt() *BasePrompt {
	return &BasePrompt{
		name:        "test_generation",
		description: "Generate comprehensive test cases for code",
		content: `You are a testing expert. Generate comprehensive test cases including:
1. Unit tests for individual functions
2. Edge cases and boundary conditions
3. Error handling scenarios
4. Integration test suggestions
5. Performance test scenarios where relevant

Use the appropriate testing framework for the language.`,
		arguments: []types.PromptArgument{
			{
				Name:        "code",
				Description: "The code to test",
				Required:    true,
			},
			{
				Name:        "framework",
				Description: "Testing framework to use",
				Required:    false,
			},
		},
	}
}

// NewDocumentationPrompt creates a documentation generation prompt
func NewDocumentationPrompt() *BasePrompt {
	return &BasePrompt{
		name:        "documentation",
		description: "Generate comprehensive documentation for code",
		content: `You are a technical documentation expert. Generate clear documentation including:
1. Function/method descriptions
2. Parameter explanations
3. Return value descriptions
4. Usage examples
5. Important notes and warnings

Follow the language's documentation conventions.`,
		arguments: []types.PromptArgument{
			{
				Name:        "code",
				Description: "The code to document",
				Required:    true,
			},
			{
				Name:        "style",
				Description: "Documentation style guide to follow",
				Required:    false,
			},
		},
	}
}
