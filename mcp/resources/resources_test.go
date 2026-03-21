package resources

import (
	"strings"
	"testing"

	"github.com/termfx/morfx/mcp/types"
)

func TestBaseResource_Interface(t *testing.T) {
	resource := &BaseResource{
		name:        "test_resource",
		description: "Test resource",
		uri:         "test://resource",
		mimeType:    "text/plain",
		contentFunc: func() (string, error) { return "test content", nil },
	}

	// Test interface compliance
	var _ types.Resource = resource

	if resource.Name() != "test_resource" {
		t.Errorf("Expected name 'test_resource', got '%s'", resource.Name())
	}

	if resource.Description() != "Test resource" {
		t.Errorf("Expected description 'Test resource', got '%s'", resource.Description())
	}

	if resource.URI() != "test://resource" {
		t.Errorf("Expected URI 'test://resource', got '%s'", resource.URI())
	}

	if resource.MimeType() != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got '%s'", resource.MimeType())
	}

	content, err := resource.Contents()
	if err != nil {
		t.Errorf("Contents() failed: %v", err)
	}

	if content != "test content" {
		t.Errorf("Expected content 'test content', got '%s'", content)
	}
}

func TestBaseResource_NoContentFunc(t *testing.T) {
	resource := &BaseResource{
		name: "no_content",
		uri:  "test://no-content",
	}

	_, err := resource.Contents()
	if err == nil {
		t.Error("Expected error when no content function defined")
	}
}

func TestResourceRegistry_Register(t *testing.T) {
	registry := &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}

	resource := &BaseResource{
		name: "test_resource",
		uri:  "test://resource",
	}

	registry.Register("test", resource)

	if len(registry.resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(registry.resources))
	}

	if len(registry.ordered) != 1 {
		t.Errorf("Expected 1 ordered resource, got %d", len(registry.ordered))
	}
}

func TestResourceRegistry_Get(t *testing.T) {
	registry := &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}

	resource := &BaseResource{name: "test_resource"}
	registry.Register("test", resource)

	retrieved, exists := registry.Get("test")
	if !exists {
		t.Error("Resource should exist")
	}

	if retrieved.Name() != "test_resource" {
		t.Errorf("Expected name 'test_resource', got '%s'", retrieved.Name())
	}

	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Non-existent resource should not exist")
	}
}

func TestResourceRegistry_List(t *testing.T) {
	registry := &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}

	resource1 := &BaseResource{name: "resource1"}
	resource2 := &BaseResource{name: "resource2"}

	registry.Register("first", resource1)
	registry.Register("second", resource2)

	resources := registry.List()
	if len(resources) != 2 {
		t.Errorf("Expected 2 resources, got %d", len(resources))
	}

	if resources[0].Name() != "resource1" {
		t.Errorf("Expected first resource 'resource1', got '%s'", resources[0].Name())
	}
}

func TestNewSupportedLanguagesResource(t *testing.T) {
	resource := NewSupportedLanguagesResource()

	if resource.Name() != "Supported Languages" {
		t.Errorf("Expected name 'Supported Languages', got '%s'", resource.Name())
	}

	if resource.URI() != "config://supported-languages" {
		t.Errorf("Expected URI 'config://supported-languages', got '%s'", resource.URI())
	}

	if resource.MimeType() != "application/json" {
		t.Errorf("Expected MIME type 'application/json', got '%s'", resource.MimeType())
	}

	content, err := resource.Contents()
	if err != nil {
		t.Errorf("Contents() failed: %v", err)
	}

	if !strings.Contains(content, "languages") {
		t.Error("Content should contain 'languages'")
	}

	if !strings.Contains(content, "go") {
		t.Error("Content should contain 'go' language")
	}
}

func TestNewTransformationMethodsResource(t *testing.T) {
	resource := NewTransformationMethodsResource()

	if resource.Name() != "Transformation Methods" {
		t.Errorf("Expected name 'Transformation Methods', got '%s'", resource.Name())
	}

	content, err := resource.Contents()
	if err != nil {
		t.Errorf("Contents() failed: %v", err)
	}

	if !strings.Contains(content, "methods") {
		t.Error("Content should contain 'methods'")
	}

	if !strings.Contains(content, "replace") {
		t.Error("Content should contain 'replace' method")
	}
}

func TestNewReadmeResource(t *testing.T) {
	resource := NewReadmeResource()

	if resource.Name() != "README" {
		t.Errorf("Expected name 'README', got '%s'", resource.Name())
	}

	if resource.MimeType() != "text/markdown" {
		t.Errorf("Expected MIME type 'text/markdown', got '%s'", resource.MimeType())
	}

	content, err := resource.Contents()
	if err != nil {
		t.Errorf("Contents() failed: %v", err)
	}

	if !strings.Contains(content, "Morfx") {
		t.Error("Content should contain 'Morfx'")
	}
}

func TestNewAPIDocResource(t *testing.T) {
	resource := NewAPIDocResource()

	if resource.Name() != "API Documentation" {
		t.Errorf("Expected name 'API Documentation', got '%s'", resource.Name())
	}

	content, err := resource.Contents()
	if err != nil {
		t.Errorf("Contents() failed: %v", err)
	}

	if !strings.Contains(content, "API Documentation") {
		t.Error("Content should contain 'API Documentation'")
	}

	if !strings.Contains(content, "Query Tool") {
		t.Error("Content should contain 'Query Tool'")
	}
}

func TestRegisterAll(t *testing.T) {
	// Clear registry for clean test
	Registry = &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}

	RegisterAll()

	resources := Registry.List()
	if len(resources) != 4 {
		t.Errorf("Expected 4 registered resources, got %d", len(resources))
	}

	expectedResources := []string{
		"config://supported-languages",
		"config://transformation-methods",
		"docs://readme",
		"docs://api",
	}

	for _, expected := range expectedResources {
		_, exists := Registry.Get(expected)
		if !exists {
			t.Errorf("Expected resource '%s' to be registered", expected)
		}
	}
}

func TestGet(t *testing.T) {
	// Clear and initialize registry
	Registry = &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}
	RegisterAll()

	resource, exists := Get("config://supported-languages")
	if !exists {
		t.Error("supported-languages resource should exist")
	}

	if resource.Name() != "Supported Languages" {
		t.Errorf("Expected name 'Supported Languages', got '%s'", resource.Name())
	}
}

func TestGetDefinitions(t *testing.T) {
	// Clear and initialize registry
	Registry = &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
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

	if def.URI == "" {
		t.Error("Definition URI should not be empty")
	}

	if def.MimeType == "" {
		t.Error("Definition MimeType should not be empty")
	}
}

func TestInit(t *testing.T) {
	// Clear registry
	Registry = &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}

	Init()

	resources := Registry.List()
	if len(resources) == 0 {
		t.Error("Init should register resources")
	}
}

func TestResourceRegistry_DuplicateRegistration(t *testing.T) {
	registry := &resourceRegistry{
		resources: make(map[string]types.Resource),
		ordered:   make([]string, 0),
	}

	resource1 := &BaseResource{name: "first"}
	resource2 := &BaseResource{name: "second"}

	registry.Register("test", resource1)
	registry.Register("test", resource2) // Duplicate key

	// Should have only one entry in ordered list
	if len(registry.ordered) != 1 {
		t.Errorf("Expected 1 ordered entry, got %d", len(registry.ordered))
	}

	// Should overwrite the resource
	retrieved, _ := registry.Get("test")
	if retrieved.Name() != "second" {
		t.Errorf("Expected overwritten resource 'second', got '%s'", retrieved.Name())
	}
}
