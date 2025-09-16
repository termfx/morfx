package resources

import (
	"fmt"
	"os"
	"sync"

	"github.com/termfx/morfx/mcp/types"
)

// BaseResource provides common resource functionality
type BaseResource struct {
	name        string
	description string
	uri         string
	mimeType    string
	contentFunc func() (string, error)
}

// Name returns the resource name
func (r *BaseResource) Name() string {
	return r.name
}

// Description returns the resource description
func (r *BaseResource) Description() string {
	return r.description
}

// URI returns the resource URI
func (r *BaseResource) URI() string {
	return r.uri
}

// MimeType returns the resource MIME type
func (r *BaseResource) MimeType() string {
	return r.mimeType
}

// Contents returns the resource contents
func (r *BaseResource) Contents() (string, error) {
	if r.contentFunc != nil {
		return r.contentFunc()
	}
	return "", fmt.Errorf("no content function defined")
}

// resourceRegistry manages resources internally
type resourceRegistry struct {
	mu        sync.RWMutex
	resources map[string]types.Resource
	ordered   []string
}

// Registry holds all registered resources
var Registry = &resourceRegistry{
	resources: make(map[string]types.Resource),
	ordered:   make([]string, 0),
}

// Register adds a resource to the registry
func (r *resourceRegistry) Register(name string, resource types.Resource) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.resources[name]; !exists {
		r.ordered = append(r.ordered, name)
	}
	r.resources[name] = resource
}

// Get retrieves a resource by name
func (r *resourceRegistry) Get(name string) (types.Resource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resource, exists := r.resources[name]
	return resource, exists
}

// List returns all resources in registration order
func (r *resourceRegistry) List() []types.Resource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]types.Resource, 0, len(r.ordered))
	for _, name := range r.ordered {
		result = append(result, r.resources[name])
	}
	return result
}

// Init initializes the resource registry
func Init() {
	RegisterAll()
}

// RegisterAll registers all built-in resources
func RegisterAll() {
	// Configuration resources
	Registry.Register("config://supported-languages", NewSupportedLanguagesResource())
	Registry.Register("config://transformation-methods", NewTransformationMethodsResource())

	// Documentation resources
	Registry.Register("docs://readme", NewReadmeResource())
	Registry.Register("docs://api", NewAPIDocResource())
}

// Get retrieves a resource by name
func Get(name string) (types.Resource, bool) {
	return Registry.Get(name)
}

// GetDefinitions returns all resource definitions
func GetDefinitions() []types.ResourceDefinition {
	resources := Registry.List()
	definitions := make([]types.ResourceDefinition, 0, len(resources))

	for _, resource := range resources {
		definitions = append(definitions, types.ResourceDefinition{
			URI:         resource.URI(),
			Name:        resource.Name(),
			Description: resource.Description(),
			MimeType:    resource.MimeType(),
		})
	}

	return definitions
} // NewSupportedLanguagesResource creates a resource listing supported languages
func NewSupportedLanguagesResource() *BaseResource {
	return &BaseResource{
		name:        "Supported Languages",
		description: "List of programming languages supported by Morfx",
		uri:         "config://supported-languages",
		mimeType:    "application/json",
		contentFunc: func() (string, error) {
			return `{
  "languages": [
    {"name": "go", "extensions": [".go"], "features": ["full"]},
    {"name": "python", "extensions": [".py"], "features": ["full"]},
    {"name": "javascript", "extensions": [".js", ".jsx"], "features": ["full"]},
    {"name": "typescript", "extensions": [".ts", ".tsx"], "features": ["full"]},
    {"name": "php", "extensions": [".php"], "features": ["full"]}
  ]
}`, nil
		},
	}
}

// NewTransformationMethodsResource creates a resource listing transformation methods
func NewTransformationMethodsResource() *BaseResource {
	return &BaseResource{
		name:        "Transformation Methods",
		description: "Available code transformation methods",
		uri:         "config://transformation-methods",
		mimeType:    "application/json",
		contentFunc: func() (string, error) {
			return `{
  "methods": [
    {"name": "replace", "description": "Replace code elements"},
    {"name": "delete", "description": "Delete code elements"},
    {"name": "insert_before", "description": "Insert code before elements"},
    {"name": "insert_after", "description": "Insert code after elements"},
    {"name": "append", "description": "Append code to elements"}
  ]
}`, nil
		},
	}
}

// NewReadmeResource creates a resource for the README
func NewReadmeResource() *BaseResource {
	return &BaseResource{
		name:        "README",
		description: "Morfx documentation and usage guide",
		uri:         "docs://readme",
		mimeType:    "text/markdown",
		contentFunc: func() (string, error) {
			// Try to read actual README if available
			content, err := os.ReadFile("README.md")
			if err != nil {
				return "# Morfx\n\nDeterministic AST-based code transformations for AI agents.", nil
			}
			return string(content), nil
		},
	}
}

// NewAPIDocResource creates a resource for API documentation
func NewAPIDocResource() *BaseResource {
	return &BaseResource{
		name:        "API Documentation",
		description: "Complete API reference for Morfx tools",
		uri:         "docs://api",
		mimeType:    "text/markdown",
		contentFunc: func() (string, error) {
			return `# Morfx API Documentation

## Query Tool
Find code elements using natural language queries.

## Replace Tool
Replace code elements with new implementations.

## Delete Tool
Remove code elements from source files.

## Insert Tools
Insert code before or after specific elements.

## Append Tool
Append code to functions, classes, or files.
`, nil
		},
	}
}
