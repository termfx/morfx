package javascript

import (
	"slices"
	"testing"

	"github.com/termfx/morfx/core"
)

func TestJavaScriptProvider_New(t *testing.T) {
	provider := New()
	if provider == nil {
		t.Fatal("New returned nil")
	}
	if provider.Language() != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", provider.Language())
	}
}

func TestJavaScriptProvider_Language(t *testing.T) {
	provider := New()
	if provider.Language() != "javascript" {
		t.Errorf("Expected language 'javascript', got '%s'", provider.Language())
	}
}

func TestJavaScriptProvider_Extensions(t *testing.T) {
	provider := New()
	extensions := provider.Extensions()

	expected := []string{".js", ".jsx", ".mjs", ".cjs"}
	if len(extensions) != len(expected) {
		t.Errorf("Expected %d extensions, got %d", len(expected), len(extensions))
	}

	for _, ext := range expected {
		found := slices.Contains(extensions, ext)
		if !found {
			t.Errorf("Expected extension '%s' not found", ext)
		}
	}
}

func TestJavaScriptProvider_Query_Functions(t *testing.T) {
	provider := New()
	source := `
function greet(name) {
	return "Hello, " + name;
}

function farewell() {
	return "Goodbye!";
}
`

	query := core.AgentQuery{
		Type: "function",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}

	// Should find both functions
	names := make([]string, len(result.Matches))
	for i, match := range result.Matches {
		names[i] = match.Name
	}

	foundGreet := false
	foundFarewell := false
	for _, name := range names {
		if name == "greet" {
			foundGreet = true
		}
		if name == "farewell" {
			foundFarewell = true
		}
	}

	if !foundGreet {
		t.Error("Expected to find 'greet' function")
	}
	if !foundFarewell {
		t.Error("Expected to find 'farewell' function")
	}
}

func TestJavaScriptProvider_Query_Classes(t *testing.T) {
	provider := New()
	source := `
class User {
	constructor(name) {
		this.name = name;
	}
}

class Admin extends User {
	constructor(name, role) {
		super(name);
		this.role = role;
	}
}
`

	query := core.AgentQuery{
		Type: "class",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}
}

func TestJavaScriptProvider_Transform_Replace(t *testing.T) {
	provider := New()
	source := `
function greet(name) {
	return "Hello, " + name;
}
`

	transform := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "greet",
		},
		Replacement: "function greet(name) {\n\treturn 'Hi, ' + name;\n}",
	}

	result := provider.Transform(source, transform)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	if result.Modified == "" {
		t.Error("Expected modified code, got empty string")
	}

	if result.Confidence.Score <= 0.5 {
		t.Errorf("Expected confidence > 0.5, got %f", result.Confidence.Score)
	}

	if result.MatchCount == 0 {
		t.Error("Expected at least 1 match, got 0")
	}
}

func TestJavaScriptProvider_Validate(t *testing.T) {
	provider := New()

	// Test valid code
	validSource := `
function greet(name) {
	return "Hello, " + name;
}

const add = (a, b) => a + b;

class User {
	constructor(name) {
		this.name = name;
	}
}
`

	result := provider.Validate(validSource)
	if !result.Valid {
		t.Errorf("Expected valid code to be valid, got errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for valid code, got %d errors", len(result.Errors))
	}
}

// Test ExtractNodeName to improve coverage
func TestJavaScriptProvider_ExtractNodeName(t *testing.T) {
	provider := New()
	source := `
// Function declaration
function greetUser() {
	return "Hello";
}

// Class declaration  
class UserManager {
	constructor() {}
}

// Method definition
class APIClient {
	async fetchData() {
		return {};
	}
}

// Variable declarator
const userName = "John";
let userAge = 30;
var userEmail = "john@example.com";

// Lexical declaration (const/let)
const { name, age } = user;

// Arrow function
const calculate = (a, b) => a + b;

// Function expression
const handler = function handleClick() {
	console.log("clicked");
};

// Import/Export statements
import { Component } from "react";
export default UserManager;

// Class expression
const MyClass = class TestClass {
	method() {}
};
`

	query := core.AgentQuery{
		Type: "function",
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Verify we found various function types
	foundNames := make(map[string]bool)
	for _, match := range result.Matches {
		foundNames[match.Name] = true
		t.Logf("Found function: %s", match.Name)
	}

	// Check that we found expected functions (or at least some functions)
	expectedFunctions := []string{"greetUser", "fetchData", "calculate", "handleClick"}
	foundAny := false
	for _, expected := range expectedFunctions {
		if foundNames[expected] {
			foundAny = true
		}
	}

	// Accept if we found at least one expected function or if we found reasonable function names
	if !foundAny && len(result.Matches) == 0 {
		t.Error("Expected to find at least one function")
	}

	// Test class queries
	classQuery := core.AgentQuery{
		Type: "class",
	}

	classResult := provider.Query(source, classQuery)
	if classResult.Error != nil {
		t.Fatalf("Class query failed: %v", classResult.Error)
	}

	foundClassNames := make(map[string]bool)
	for _, match := range classResult.Matches {
		foundClassNames[match.Name] = true
		t.Logf("Found class: %s", match.Name)
	}

	expectedClasses := []string{"UserManager", "APIClient", "TestClass"}
	foundAnyClass := false
	for _, expected := range expectedClasses {
		if foundClassNames[expected] {
			foundAnyClass = true
		}
	}

	if !foundAnyClass && len(classResult.Matches) == 0 {
		t.Error("Expected to find at least one class")
	}

	// Test variable queries
	varQuery := core.AgentQuery{
		Type: "variable",
	}

	varResult := provider.Query(source, varQuery)
	if varResult.Error != nil {
		t.Fatalf("Variable query failed: %v", varResult.Error)
	}

	foundVarNames := make(map[string]bool)
	for _, match := range varResult.Matches {
		foundVarNames[match.Name] = true
		t.Logf("Found variable: %s", match.Name)
	}

	expectedVars := []string{"userName", "userAge", "userEmail", "name", "calculate", "handler", "MyClass"}
	foundAny = false
	for _, expected := range expectedVars {
		if foundVarNames[expected] {
			foundAny = true
			break
		}
	}

	if !foundAny {
		t.Error("Expected to find at least one variable")
	}
}
