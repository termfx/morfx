# Creating Language Providers

This guide walks you through creating a new language provider for morfx. By implementing the [`LanguageProvider`](../../internal/provider/contract.go:26) interface, you can add support for any programming language with a Tree-sitter parser.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Step-by-Step Guide](#step-by-step-guide)
- [Interface Methods Reference](#interface-methods-reference)
- [Example: Ruby Provider](#example-ruby-provider)
- [NodeKind Mapping Strategy](#nodekind-mapping-strategy)
- [Tree-sitter Query Templates](#tree-sitter-query-templates)
- [Testing Your Provider](#testing-your-provider)
- [Registering Your Provider](#registering-your-provider)
- [Best Practices](#best-practices)
- [Advanced Topics](#advanced-topics)

## Overview

A language provider acts as a bridge between morfx's universal abstractions and a specific programming language's AST structure. It translates universal concepts like "function" or "class" into language-specific Tree-sitter queries and node types.

```
Universal Query: "func:Test*"
        ↓
Ruby Provider Translation: "(method name: (identifier) @name) (#match? @name \"Test.*\")"
        ↓
Tree-sitter Query Execution
        ↓
Universal Results
```

## Prerequisites

Before creating a provider, ensure you have:

1. **Go 1.24+** installed
2. **Tree-sitter grammar** for your target language
3. **Go bindings** for the Tree-sitter grammar
4. **Basic understanding** of Tree-sitter query syntax
5. **Knowledge** of your target language's AST structure

## Step-by-Step Guide

### Step 1: Set Up the Provider Package

Create a new directory under [`internal/lang/`](../../internal/lang/) for your language:

```bash
mkdir internal/lang/ruby
cd internal/lang/ruby
```

Create the main provider file:

```bash
touch provider.go
```

### Step 2: Define the Provider Struct

Start with the basic provider structure:

```go
package ruby

import (
    "fmt"
    "strings"

    sitter "github.com/smacker/go-tree-sitter"
    ruby_sitter "github.com/smacker/go-tree-sitter/ruby"

    "github.com/termfx/morfx/internal/core"
    "github.com/termfx/morfx/internal/provider"
)

// RubyProvider implements the LanguageProvider interface for Ruby language support
type RubyProvider struct {
    provider.BaseProvider
    dslVocabulary map[string]core.NodeKind
}

// NewProvider creates a new instance of the Ruby language provider
func NewProvider() provider.LanguageProvider {
    p := &RubyProvider{}
    p.Initialize()
    return p
}
```

### Step 3: Implement Metadata Methods

These methods provide basic information about your language:

```go
// Lang returns the canonical name of the language
func (p *RubyProvider) Lang() string {
    return "ruby"
}

// Aliases returns alternative names for this language
func (p *RubyProvider) Aliases() []string {
    return []string{"ruby", "rb"}
}

// Extensions returns file extensions for this language
func (p *RubyProvider) Extensions() []string {
    return []string{".rb", ".ruby"}
}

// GetSitterLanguage returns the Tree-sitter language for Ruby
func (p *RubyProvider) GetSitterLanguage() *sitter.Language {
    return ruby_sitter.GetLanguage()
}
```

### Step 4: Set Up DSL Vocabulary

Map language-specific terms to universal kinds:

```go
func (p *RubyProvider) Initialize() {
    // Define Ruby-specific DSL vocabulary
    p.dslVocabulary = map[string]core.NodeKind{
        // Ruby-specific terms
        "def":     core.KindFunction,    // Ruby method definition
        "class":   core.KindClass,       // Ruby class
        "module":  core.KindClass,       // Ruby module (treated as class)
        "require": core.KindImport,      // Ruby require
        "attr":    core.KindField,       // Ruby attribute
        // Universal terms (for compatibility)
        "function": core.KindFunction,
        "method":   core.KindMethod,
        "import":   core.KindImport,
    }

    // Define NodeMapping for universal kinds to Ruby AST
    mappings := []provider.NodeMapping{
        // Add mappings here (see Step 5)
    }

    p.BuildMappings(mappings)
}
```

### Step 5: Create NodeKind Mappings

This is the heart of the provider - mapping universal concepts to language-specific AST patterns:

```go
func (p *RubyProvider) Initialize() {
    // ... dslVocabulary setup

    mappings := []provider.NodeMapping{
        {
            Kind:        core.KindFunction,
            NodeTypes:   []string{"method"},
            NameCapture: "@name",
            Template:    `(method name: (identifier) @name %s)`,
        },
        {
            Kind:        core.KindClass,
            NodeTypes:   []string{"class"},
            NameCapture: "@name",
            Template:    `(class name: (constant) @name %s)`,
        },
        {
            Kind:        core.KindVariable,
            NodeTypes:   []string{"assignment"},
            NameCapture: "@name",
            Template:    `(assignment left: (identifier) @name %s)`,
        },
        {
            Kind:        core.KindImport,
            NodeTypes:   []string{"call"},
            NameCapture: "@arg",
            Template:    `(call method: (identifier) @method arguments: (argument_list (string) @arg) %s (#eq? @method "require"))`,
        },
        {
            Kind:        core.KindCall,
            NodeTypes:   []string{"call"},
            NameCapture: "@method",
            Template:    `(call method: (identifier) @method %s)`,
        },
        {
            Kind:        core.KindCondition,
            NodeTypes:   []string{"if", "unless", "case"},
            NameCapture: "@condition",
            Template:    `[(if) (unless) (case)] @condition %s`,
        },
        {
            Kind:        core.KindLoop,
            NodeTypes:   []string{"while", "for"},
            NameCapture: "@loop",
            Template:    `[(while) (for)] @loop %s`,
        },
    }

    p.BuildMappings(mappings)
}
```

### Step 6: Implement Translation Methods

Implement the core translation methods:

```go
// NormalizeDSLKind translates Ruby-specific DSL terms to universal kinds
func (p *RubyProvider) NormalizeDSLKind(dslKind string) core.NodeKind {
    if universalKind, exists := p.dslVocabulary[dslKind]; exists {
        return universalKind
    }
    return core.NodeKind(dslKind)
}

// TranslateQuery translates a universal query to Ruby-specific Tree-sitter query
func (p *RubyProvider) TranslateQuery(q *core.Query) (string, error) {
    kind := p.NormalizeDSLKind(string(q.Kind))
    
    mappings := p.TranslateKind(kind)
    if len(mappings) == 0 {
        return "", fmt.Errorf("unsupported node kind: %s", q.Kind)
    }

    var queries []string
    for _, mapping := range mappings {
        query := p.BuildQueryFromMapping(mapping, q)
        if query != "" {
            queries = append(queries, query)
        }
    }

    if len(queries) == 0 {
        return "", fmt.Errorf("no valid queries generated for kind: %s", q.Kind)
    }

    return strings.Join(queries, "\n"), nil
}
```

### Step 7: Implement AST Introspection Methods

These methods extract information from Tree-sitter nodes:

```go
// GetNodeKind determines the universal kind of a Tree-sitter node
func (p *RubyProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
    switch node.Type() {
    case "method":
        return core.KindFunction
    case "class":
        return core.KindClass
    case "module":
        return core.KindClass
    case "assignment":
        return core.KindVariable
    case "call":
        // Check if it's a require call (import) or regular call
        if p.isRequireCall(node) {
            return core.KindImport
        }
        return core.KindCall
    case "if", "unless", "case":
        return core.KindCondition
    case "while", "for":
        return core.KindLoop
    default:
        return core.NodeKind(node.Type())
    }
}

// GetNodeName extracts the name/identifier from a Tree-sitter node
func (p *RubyProvider) GetNodeName(node *sitter.Node, source []byte) string {
    switch node.Type() {
    case "method":
        if nameChild := node.ChildByFieldName("name"); nameChild != nil {
            return nameChild.Content(source)
        }
    case "class", "module":
        if nameChild := node.ChildByFieldName("name"); nameChild != nil {
            return nameChild.Content(source)
        }
    case "assignment":
        if leftChild := node.ChildByFieldName("left"); leftChild != nil {
            return leftChild.Content(source)
        }
    case "call":
        if p.isRequireCall(node) {
            // Extract the required file/module name
            return p.extractRequireArgument(node, source)
        }
        if methodChild := node.ChildByFieldName("method"); methodChild != nil {
            return methodChild.Content(source)
        }
    }
    return node.Content(source)
}

// Helper method to check if a call is a require
func (p *RubyProvider) isRequireCall(node *sitter.Node) bool {
    if methodChild := node.ChildByFieldName("method"); methodChild != nil {
        methodName := methodChild.Content([]byte{}) // We'll get proper source elsewhere
        return methodName == "require" || methodName == "require_relative"
    }
    return false
}

// Helper method to extract require argument
func (p *RubyProvider) extractRequireArgument(node *sitter.Node, source []byte) string {
    if argsChild := node.ChildByFieldName("arguments"); argsChild != nil {
        if argsChild.ChildCount() > 0 {
            firstArg := argsChild.Child(0)
            content := firstArg.Content(source)
            // Remove quotes from string literals
            if len(content) >= 2 && (content[0] == '"' || content[0] == '\'') {
                return content[1 : len(content)-1]
            }
            return content
        }
    }
    return ""
}
```

### Step 8: Implement Scope Detection

Implement scope-related methods:

```go
// GetNodeScope determines the scope type for a given node
func (p *RubyProvider) GetNodeScope(node *sitter.Node) core.ScopeType {
    switch node.Type() {
    case "program":
        return core.ScopeFile
    case "class", "module":
        return core.ScopeClass
    case "method":
        return core.ScopeFunction
    case "begin", "if", "unless", "while", "for", "case":
        return core.ScopeBlock
    default:
        // Use base implementation for fallback
        return p.BaseProvider.GetNodeScope(node)
    }
}

// FindEnclosingScope can use the base implementation or override for Ruby-specific logic
// The BaseProvider implementation works well for most languages
```

### Step 9: Implement Language Services (Optional)

These methods provide additional language-specific functionality:

```go
// OrganizeImports organizes Ruby require statements
func (p *RubyProvider) OrganizeImports(source []byte) ([]byte, error) {
    // Ruby-specific import organization logic
    // For now, return unchanged
    return source, nil
}

// Format formats Ruby code (could integrate with rubocop or similar)
func (p *RubyProvider) Format(source []byte) ([]byte, error) {
    // Ruby-specific formatting logic
    // For now, return unchanged
    return source, nil
}

// QuickCheck performs basic syntax validation for Ruby
func (p *RubyProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
    var diagnostics []core.QuickCheckDiagnostic
    
    // Parse and check for syntax errors
    parser := sitter.NewParser()
    parser.SetLanguage(ruby_sitter.GetLanguage())
    tree, err := parser.ParseCtx(nil, nil, source)
    if err != nil {
        diagnostics = append(diagnostics, core.QuickCheckDiagnostic{
            Severity: "error",
            Message:  fmt.Sprintf("Parse error: %v", err),
            Line:     1,
            Column:   1,
        })
        return diagnostics
    }
    defer tree.Close()
    
    // Check for ERROR nodes in the AST
    p.checkForParseErrors(tree.RootNode(), source, &diagnostics)
    
    return diagnostics
}

func (p *RubyProvider) checkForParseErrors(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
    if node.Type() == "ERROR" {
        startPoint := node.StartPoint()
        *diagnostics = append(*diagnostics, core.QuickCheckDiagnostic{
            Severity: "error",
            Message:  "Syntax error",
            Line:     int(startPoint.Row) + 1,
            Column:   int(startPoint.Column) + 1,
        })
    }
    
    for i := 0; i < int(node.ChildCount()); i++ {
        p.checkForParseErrors(node.Child(i), source, diagnostics)
    }
}
```

## Interface Methods Reference

### Required Methods

#### Metadata Methods
- **`Lang() string`**: Return canonical language name (e.g., "ruby")
- **`Aliases() []string`**: Return alternative names (e.g., ["ruby", "rb"])
- **`Extensions() []string`**: Return file extensions (e.g., [".rb"])
- **`GetSitterLanguage() *sitter.Language`**: Return Tree-sitter language parser

#### Translation Methods
- **`TranslateKind(kind core.NodeKind) []NodeMapping`**: Map universal kind to language-specific mappings
- **`TranslateQuery(q *core.Query) (string, error)`**: Convert universal query to Tree-sitter query

#### AST Introspection Methods
- **`GetNodeKind(node *sitter.Node) core.NodeKind`**: Determine universal kind for AST node
- **`GetNodeName(node *sitter.Node, source []byte) string`**: Extract identifier/name from AST node
- **`ParseAttributes(node *sitter.Node, source []byte) map[string]string`**: Extract metadata as key-value pairs

#### Scope Detection Methods
- **`GetNodeScope(node *sitter.Node) core.ScopeType`**: Determine scope type for node
- **`FindEnclosingScope(node *sitter.Node, scope core.ScopeType) *sitter.Node`**: Find nearest enclosing scope

#### Language Services Methods
- **`NormalizeDSLKind(dslKind string) core.NodeKind`**: Map language-specific DSL terms
- **`IsBlockLevelNode(nodeType string) bool`**: Determine if node should be treated as block-level
- **`GetDefaultIgnorePatterns() ([]string, []string)`**: Return default file and symbol ignore patterns
- **`OrganizeImports(source []byte) ([]byte, error)`**: Organize import statements
- **`Format(source []byte) ([]byte, error)`**: Format source code
- **`QuickCheck(source []byte) []core.QuickCheckDiagnostic`**: Perform quick syntax validation

## Example: Ruby Provider

Here's a complete minimal Ruby provider implementation:

```go
package ruby

import (
    "fmt"
    "strings"

    sitter "github.com/smacker/go-tree-sitter"
    ruby_sitter "github.com/smacker/go-tree-sitter/ruby"

    "github.com/termfx/morfx/internal/core"
    "github.com/termfx/morfx/internal/provider"
)

type RubyProvider struct {
    provider.BaseProvider
    dslVocabulary map[string]core.NodeKind
}

func NewProvider() provider.LanguageProvider {
    p := &RubyProvider{}
    p.Initialize()
    return p
}

func (p *RubyProvider) Initialize() {
    p.dslVocabulary = map[string]core.NodeKind{
        "def":     core.KindFunction,
        "class":   core.KindClass,
        "module":  core.KindClass,
        "require": core.KindImport,
        "attr":    core.KindField,
    }

    mappings := []provider.NodeMapping{
        {
            Kind:        core.KindFunction,
            NodeTypes:   []string{"method"},
            NameCapture: "@name",
            Template:    `(method name: (identifier) @name %s)`,
        },
        {
            Kind:        core.KindClass,
            NodeTypes:   []string{"class"},
            NameCapture: "@name", 
            Template:    `(class name: (constant) @name %s)`,
        },
    }

    p.BuildMappings(mappings)
}

func (p *RubyProvider) Lang() string { return "ruby" }
func (p *RubyProvider) Aliases() []string { return []string{"ruby", "rb"} }
func (p *RubyProvider) Extensions() []string { return []string{".rb"} }
func (p *RubyProvider) GetSitterLanguage() *sitter.Language { return ruby_sitter.GetLanguage() }

func (p *RubyProvider) NormalizeDSLKind(dslKind string) core.NodeKind {
    if kind, exists := p.dslVocabulary[dslKind]; exists {
        return kind
    }
    return core.NodeKind(dslKind)
}

func (p *RubyProvider) TranslateQuery(q *core.Query) (string, error) {
    kind := p.NormalizeDSLKind(string(q.Kind))
    mappings := p.TranslateKind(kind)
    if len(mappings) == 0 {
        return "", fmt.Errorf("unsupported node kind: %s", q.Kind)
    }

    var queries []string
    for _, mapping := range mappings {
        query := p.BuildQueryFromMapping(mapping, q)
        if query != "" {
            queries = append(queries, query)
        }
    }

    return strings.Join(queries, "\n"), nil
}

func (p *RubyProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
    switch node.Type() {
    case "method": return core.KindFunction
    case "class": return core.KindClass
    default: return core.NodeKind(node.Type())
    }
}

func (p *RubyProvider) GetNodeName(node *sitter.Node, source []byte) string {
    switch node.Type() {
    case "method":
        if name := node.ChildByFieldName("name"); name != nil {
            return name.Content(source)
        }
    case "class":
        if name := node.ChildByFieldName("name"); name != nil {
            return name.Content(source)
        }
    }
    return node.Content(source)
}
```

## NodeKind Mapping Strategy

### Understanding Tree-sitter AST Structure

Before creating mappings, examine your language's Tree-sitter grammar:

1. **Install tree-sitter CLI**: `npm install -g tree-sitter-cli`
2. **Parse sample code**: `tree-sitter parse example.rb`
3. **Study the AST structure** to understand node types and field names

Example Ruby AST for `def hello; end`:
```
program [0, 0] - [0, 15]
  method [0, 0] - [0, 15]
    name: identifier [0, 4] - [0, 9]
```

### Mapping Universal Kinds

Map each universal [`NodeKind`](../../internal/core/contracts.go:10) to appropriate language constructs:

| Universal Kind | Ruby Equivalent | Tree-sitter Node Type |
|----------------|-----------------|----------------------|
| `KindFunction` | `def method_name` | `method` |
| `KindClass` | `class ClassName` | `class` |
| `KindVariable` | `variable = value` | `assignment` |
| `KindImport` | `require "file"` | `call` (with method="require") |
| `KindCall` | `object.method()` | `call` |
| `KindCondition` | `if condition` | `if`, `unless`, `case` |
| `KindLoop` | `while condition` | `while`, `for` |

### Creating Effective Templates

Template syntax follows Tree-sitter query patterns:

- **Basic pattern**: `(node_type) @capture`
- **Field matching**: `(node_type field_name: (child_type) @capture)`
- **Multiple patterns**: `[(pattern1) (pattern2)] @capture`
- **Constraints**: `(#match? @capture "pattern")`

Examples:
```lisp
; Basic method capture
(method name: (identifier) @name %s)

; Class with superclass
(class name: (constant) @name superclass: (superclass)? %s)

; Call with method filtering
(call method: (identifier) @method %s (#eq? @method "require"))
```

## Tree-sitter Query Templates

### Template Placeholders

The `%s` placeholder in templates is replaced with constraints based on the query:

- **Pattern matching**: `(#match? @name "Test.*")` for pattern `"Test*"`
- **Type constraints**: `(#match? @type "String")` for type attribute
- **Multiple constraints**: Combined with spaces

Example template usage:
```go
Template: `(method name: (identifier) @name %s)`

Query: {Pattern: "test*", Attributes: {"visibility": "public"}}

Result: `(method name: (identifier) @name (#match? @name "test.*"))`
```

### Advanced Template Patterns

#### Handling Optional Fields
```lisp
; Optional superclass
(class name: (constant) @name superclass: (superclass)? %s)

; Optional type annotation
(assignment left: (identifier) @name right: (_) type: (type_annotation)? %s)
```

#### Multiple Node Types
```lisp
; Functions or methods
[(function_declaration) (method_declaration)] @func %s

; Different import styles
[(call method: (identifier) @method (#eq? @method "require"))
 (call method: (identifier) @method (#eq? @method "require_relative"))] %s
```

#### Nested Patterns
```lisp
; Method in class
(class 
  body: (body_statement 
    (method name: (identifier) @name %s)))

; Assignment in function
(method 
  body: (body_statement 
    (assignment left: (identifier) @var %s)))
```

## Testing Your Provider

### Unit Tests

Create comprehensive tests for your provider:

```go
package ruby

import (
    "testing"
    
    "github.com/termfx/morfx/internal/core"
    "github.com/termfx/morfx/internal/provider"
)

func TestRubyProvider_Basic(t *testing.T) {
    p := NewProvider()
    
    // Test metadata
    if p.Lang() != "ruby" {
        t.Errorf("Expected lang 'ruby', got %s", p.Lang())
    }
    
    // Test aliases
    aliases := p.Aliases()
    if len(aliases) == 0 || aliases[0] != "ruby" {
        t.Errorf("Expected aliases containing 'ruby', got %v", aliases)
    }
    
    // Test extensions
    extensions := p.Extensions()
    if len(extensions) == 0 || extensions[0] != ".rb" {
        t.Errorf("Expected extensions containing '.rb', got %v", extensions)
    }
}

func TestRubyProvider_TranslateQuery(t *testing.T) {
    p := NewProvider()
    
    tests := []struct {
        name     string
        query    *core.Query
        expected string
        wantErr  bool
    }{
        {
            name: "simple method query",
            query: &core.Query{
                Kind:    core.KindFunction,
                Pattern: "test",
            },
            expected: `(method name: (identifier) @name (#match? @name "^test$"))`,
            wantErr:  false,
        },
        {
            name: "wildcard method query",
            query: &core.Query{
                Kind:    core.KindFunction,
                Pattern: "test*",
            },
            expected: `(method name: (identifier) @name (#match? @name "^test.*$"))`,
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := p.TranslateQuery(tt.query)
            if (err != nil) != tt.wantErr {
                t.Errorf("TranslateQuery() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("TranslateQuery() = %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestRubyProvider_GetNodeKind(t *testing.T) {
    // Test with actual parsed Ruby code
    p := NewProvider()
    
    source := []byte(`
def hello
  puts "world"
end

class MyClass
  attr_reader :name
end
`)
    
    parser := sitter.NewParser()
    parser.SetLanguage(p.GetSitterLanguage())
    tree, err := parser.ParseCtx(nil, nil, source)
    if err != nil {
        t.Fatalf("Failed to parse Ruby code: %v", err)
    }
    defer tree.Close()
    
    // Find method node and test GetNodeKind
    root := tree.RootNode()
    methodNode := root.Child(0) // Should be the method
    
    kind := p.GetNodeKind(methodNode)
    if kind != core.KindFunction {
        t.Errorf("Expected KindFunction for method node, got %s", kind)
    }
    
    // Find class node and test GetNodeKind  
    classNode := root.Child(1) // Should be the class
    kind = p.GetNodeKind(classNode)
    if kind != core.KindClass {
        t.Errorf("Expected KindClass for class node, got %s", kind)
    }
}
```

### Integration Tests

Test your provider with the universal evaluator:

```go
func TestRubyProvider_Integration(t *testing.T) {
    // Create registry and register provider
    registry := registry.NewRegistry()
    provider := NewProvider()
    err := registry.RegisterProvider(provider)
    if err != nil {
        t.Fatalf("Failed to register provider: %v", err)
    }
    
    // Create evaluator with provider
    evaluator := evaluator.NewUniversalEvaluator(provider)
    
    // Test evaluation
    source := []byte(`
def test_method
  puts "hello"
end

def another_method  
  puts "world"
end
`)
    
    query := &core.Query{
        Kind:    core.KindFunction,
        Pattern: "test*",
    }
    
    results, err := evaluator.Evaluate(query, source)
    if err != nil {
        t.Fatalf("Evaluation failed: %v", err)
    }
    
    if len(results.Results) != 1 {
        t.Errorf("Expected 1 result, got %d", len(results.Results))
    }
    
    if results.Results[0].Name != "test_method" {
        t.Errorf("Expected result name 'test_method', got %s", results.Results[0].Name)
    }
}
```

### DSL Snapshot Tests

Create snapshot tests to verify query translations:

```go
func TestRubyProvider_DSLSnapshots(t *testing.T) {
    testCases := []string{
        "def:test*",
        "class:User",
        "def:initialize & class:User",
        "require:*",
        "call:puts",
    }
    
    p := NewProvider()
    parser := parser.NewUniversalParser()
    
    for _, dsl := range testCases {
        t.Run(dsl, func(t *testing.T) {
            query, err := parser.ParseQuery(dsl)
            if err != nil {
                t.Fatalf("Failed to parse DSL: %v", err)
            }
            
            tsQuery, err := p.TranslateQuery(query)
            if err != nil {
                t.Fatalf("Failed to translate query: %v", err)
            }
            
            // Compare with golden file or stored snapshot
            snapshotFile := fmt.Sprintf("testdata/snapshots/%s.snap", 
                strings.ReplaceAll(dsl, ":", "_"))
            compareWithSnapshot(t, tsQuery, snapshotFile)
        })
    }
}
```

## Registering Your Provider

### Built-in Registration

For providers included in the main binary, register in [`cmd/morfx/providers.go`](../../cmd/morfx/providers.go):

```go
package main

import (
    "github.com/termfx/morfx/internal/registry"
    
    // Import your provider
    "github.com/termfx/morfx/internal/lang/ruby"
)

func init() {
    // Register built-in providers
    registry.RegisterProvider(ruby.NewProvider())
}
```

### Plugin Registration

For external plugins, create a plugin main file:

```go
// Plugin: ruby-provider/main.go
package main

import (
    "github.com/termfx/morfx/internal/provider"
    // Your provider implementation
)

// Required: Export Provider symbol
var Provider provider.LanguageProvider = NewRubyProvider()

func NewRubyProvider() provider.LanguageProvider {
    // Return your provider instance
    return &RubyProvider{} // Your implementation
}
```

Build as plugin:
```bash
go build -buildmode=plugin -o ruby-provider.so main.go
```

Load the plugin:
```bash
# Automatic loading from ~/.morfx/plugins/
cp ruby-provider.so ~/.morfx/plugins/

# Manual loading
morfx --load-plugin ruby-provider.so
```

### Runtime Discovery

The registry automatically discovers providers:

```go
// Auto-register built-in providers and load external plugins
registry.AutoRegister()

// Get provider by various methods
provider, err := registry.GetProvider("ruby")           // By name
provider, err := registry.GetProvider("rb")             // By alias
provider, err := registry.GetProvider(".rb")            // By extension
provider, err := registry.GetProviderForFile("app.rb")  // Auto-detect
```

## Best Practices

### 1. Design for Completeness

**Map all major language constructs** to universal kinds:

```go
// Don't just handle the basics
mappings := []provider.NodeMapping{
    // Core constructs
    {Kind: core.KindFunction, NodeTypes: []string{"method", "lambda", "proc"}},
    {Kind: core.KindClass, NodeTypes: []string{"class", "module"}},
    {Kind: core.KindVariable, NodeTypes: []string{"assignment", "local_assignment"}},
    
    // Language-specific constructs
    {Kind: core.KindField, NodeTypes: []string{"attr_accessor", "attr_reader", "attr_writer"}},
    {Kind: core.KindDecorator, NodeTypes: []string{"decorator"}}, // Ruby decorators
    {Kind: core.KindImport, NodeTypes: []string{"require", "require_relative", "load"}},
}
```

### 2. Provide Comprehensive DSL Vocabulary

**Support both language-specific and universal terms**:

```go
p.dslVocabulary = map[string]core.NodeKind{
    // Language-specific (primary)
    "def":       core.KindFunction,    // Ruby method
    "class":     core.KindClass,       // Ruby class
    "module":    core.KindClass,       // Ruby module
    "require":   core.KindImport,      // Ruby require
    "attr":      core.KindField,       // Ruby attributes
    
    // Universal (compatibility)
    "function":  core.KindFunction,
    "method":    core.KindMethod,
    "import":    core.KindImport,
    "variable":  core.KindVariable,
    
    // Common aliases
    "func":      core.KindFunction,    // Go-style
    "fn":        core.KindFunction,    // Rust-style
}
```

### 3. Handle Edge Cases

**Consider language-specific edge cases**:

```go
func (p *RubyProvider) GetNodeKind(node *sitter.Node) core.NodeKind {
    switch node.Type() {
    case "call":
        // Ruby: Distinguish between method calls and requires
        if p.isRequireCall(node) {
            return core.KindImport
        }
        // Ruby: Handle metaprogramming calls as special cases
        if p.isMetaprogrammingCall(node) {
            return core.KindDecorator
        }
        return core.KindCall
        
    case "assignment":
        // Ruby: Class variables vs instance variables vs local variables
        if p.isClassVariable(node) {
            return core.KindField
        }
        return core.KindVariable
        
    case "constant":
        // Ruby: Constants are different from regular variables
        return core.KindConstant
        
    default:
        return core.NodeKind(node.Type())
    }
}
```

### 4. Implement Robust Name Extraction

**Handle different naming patterns**:

```go
func (p *RubyProvider) GetNodeName(node *sitter.Node, source []byte) string {
    switch node.Type() {
    case "method":
        // Handle method names with special characters (!, ?, =)
        if name := node.ChildByFieldName("name"); name != nil {
            return name.Content(source)
        }
        
    case "class":
        // Handle namespaced classes (Module::Class)
        if name := node.ChildByFieldName("name"); name != nil {
            return p.extractFullClassName(name, source)
        }
        
    case "assignment":
        // Handle different variable types (@instance, @@class, $global)
        if left := node.ChildByFieldName("left"); left != nil {
            return p.extractVariableName(left, source)
        }
        
    case "call":
        if p.isRequireCall(node) {
            // Extract gem/file name from require
            return p.extractRequireTarget(node, source)
        }
        // Extract method name, handling chained calls
        return p.extractCallTarget(node, source)
    }
    
    return node.Content(source)
}
```

### 5. Provide Meaningful Attributes

**Extract useful metadata**:

```go
func (p *RubyProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
    attrs := p.BaseProvider.ParseAttributes(node, source)
    
    switch node.Type() {
    case "method":
        // Ruby method visibility
        attrs["visibility"] = p.extractMethodVisibility(node, source)
        // Ruby method type (instance vs class method)
        attrs["method_type"] = p.extractMethodType(node, source)
        
    case "class":
        // Ruby superclass
        if superclass := node.ChildByFieldName("superclass"); superclass != nil {
            attrs["superclass"] = superclass.Content(source)
        }
        
    case "assignment":
        // Ruby variable scope
        attrs["scope"] = p.extractVariableScope(node, source)
        
    case "call":
        if p.isRequireCall(node) {
            // Require type (require vs require_relative)
            attrs["require_type"] = p.extractRequireType(node, source)
        }
    }
    
    return attrs
}
```

### 6. Optimize Performance

**Cache expensive operations**:

```go
type RubyProvider struct {
    provider.BaseProvider
    dslVocabulary    map[string]core.NodeKind
    nameExtractors   map[string]func(*sitter.Node, []byte) string // Cache extractors
    kindMappings     map[string]core.NodeKind                     // Cache node type mappings
}

func (p *RubyProvider) GetNodeName(node *sitter.Node, source []byte) string {
    nodeType := node.Type()
    
    // Use cached extractor if available
    if extractor, exists := p.nameExtractors[nodeType]; exists {
        return extractor(node, source)
    }
    
    // Fallback to default extraction
    return node.Content(source)
}
```

### 7. Write Comprehensive Tests

**Test all supported constructs**:

```go
func TestRubyProvider_AllConstructs(t *testing.T) {
    testCases := []struct {
        name        string
        source      string
        query       string
        expectedCount int
        expectedNames []string
    }{
        {
            name: "instance methods",
            source: `
                def hello; end
                def world!; end
                def question?; end
            `,
            query: "def:*",
            expectedCount: 3,
            expectedNames: []string{"hello", "world!", "question?"},
        },
        {
            name: "class methods",
            source: `
                class MyClass
                  def self.class_method; end
                  def instance_method; end
                end
            `,
            query: "def:*",
            expectedCount: 2,
        },
        {
            name: "modules and classes",
            source: `
                module MyModule; end
                class MyClass < BaseClass; end
            `,
            query: "class:*",
            expectedCount: 2,
        },
        {
            name: "require statements",
            source: `
                require 'json'
                require_relative 'helper'
                load 'config.rb'
            `,
            query: "require:*",
            expectedCount: 3,
        },
    }
    
    provider := NewProvider()
    evaluator := evaluator.NewUniversalEvaluator(provider)
    parser := parser.NewUniversalParser()
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            query, err := parser.ParseQuery(tc.query)
            require.NoError(t, err)
            
            results, err := evaluator.Evaluate(query, []byte(tc.source))
            require.NoError(t, err)
            
            assert.Equal(t, tc.expectedCount, len(results.Results))
            
            if len(tc.expectedNames) > 0 {
                var actualNames []string
                for _, result := range results.Results {
                    actualNames = append(actualNames, result.Name)
                }
                assert.ElementsMatch(t, tc.expectedNames, actualNames)
            }
        })
    }
}
```

## Advanced Topics

### 1. Handling Complex AST Patterns

Some languages have complex AST structures that don't map directly to universal concepts:

**Ruby Metaprogramming Example**:
```go
func (p *RubyProvider) handleMetaprogramming(node *sitter.Node, source []byte) core.NodeKind {
    // Ruby: attr_accessor, attr_reader, attr_writer create methods dynamically
    if node.Type() == "call" {
        if method := node.ChildByFieldName("method"); method != nil {
            methodName := method.Content(source)
            switch methodName {
            case "attr_accessor", "attr_reader", "attr_writer":
                return core.KindField  // These create field accessors
            case "define_method":
                return core.KindFunction  // Dynamic method definition
            case "alias_method":
                return core.KindMethod    // Method aliasing
            }
        }
    }
    return core.NodeKind(node.Type())
}
```

**JavaScript/TypeScript Decorators**:
```go
func (p *JSProvider) handleDecorators(node *sitter.Node) core.NodeKind {
    // TypeScript decorators modify classes/methods
    if node.Type() == "decorator" {
        return core.KindDecorator
    }
    
    // React Higher-Order Components
    if node.Type() == "call_expression" && p.isHOCPattern(node) {
        return core.KindDecorator
    }
    
    return core.NodeKind(node.Type())
}
```

### 2. Multi-Language File Support

Some files contain multiple languages (e.g., JSX, Vue, Svelte):

```go
func (p *JSXProvider) TranslateQuery(q *core.Query) (string, error) {
    // Handle both JavaScript and JSX constructs
    mappings := p.TranslateKind(q.Kind)
    
    var queries []string
    for _, mapping := range mappings {
        // JavaScript queries
        if jsQuery := p.buildJSQuery(mapping, q); jsQuery != "" {
            queries = append(queries, jsQuery)
        }
        
        // JSX-specific queries (components, props)
        if jsxQuery := p.buildJSXQuery(mapping, q); jsxQuery != "" {
            queries = append(queries, jsxQuery)
        }
    }
    
    return strings.Join(queries, "\n"), nil
}

func (p *JSXProvider) buildJSXQuery(mapping NodeMapping, q *core.Query) string {
    if q.Kind == core.KindClass {
        // JSX components can be function or class components
        return `(jsx_element name: (identifier) @name %s)`
    }
    return ""
}
```

### 3. Language-Specific Optimizations

**Query Caching with Language Features**:
```go
type RubyProvider struct {
    provider.BaseProvider
    
    // Ruby-specific caches
    methodCache     map[string]methodInfo
    classCache      map[string]classInfo
    requireCache    map[string][]string
}

type methodInfo struct {
    visibility   string
    methodType   string  // instance vs class
    parameters   []string
}

func (p *RubyProvider) GetNodeName(node *sitter.Node, source []byte) string {
    // Check cache first
    if cached := p.getCachedName(node); cached != "" {
        return cached
    }
    
    name := p.extractName(node, source)
    p.cacheName(node, name)
    return name
}
```

### 4. Error Recovery and Partial Parsing

Handle malformed code gracefully:

```go
func (p *RubyProvider) QuickCheck(source []byte) []core.QuickCheckDiagnostic {
    var diagnostics []core.QuickCheckDiagnostic
    
    parser := sitter.NewParser()
    parser.SetLanguage(ruby_sitter.GetLanguage())
    tree, err := parser.ParseCtx(nil, nil, source)
    if err != nil {
        return []core.QuickCheckDiagnostic{{
            Severity: "error",
            Message:  fmt.Sprintf("Parse error: %v", err),
            Line:     1,
            Column:   1,
        }}
    }
    defer tree.Close()
    
    // Check for ERROR nodes and try to provide helpful messages
    p.checkForRubySpecificErrors(tree.RootNode(), source, &diagnostics)
    
    // Check for common Ruby issues
    p.checkRubySyntaxPatterns(tree.RootNode(), source, &diagnostics)
    
    return diagnostics
}

func (p *RubyProvider) checkRubySyntaxPatterns(node *sitter.Node, source []byte, diagnostics *[]core.QuickCheckDiagnostic) {
    switch node.Type() {
    case "method":
        // Check for missing 'end'
        if !p.hasMatchingEnd(node, source) {
            p.addDiagnostic(diagnostics, node, "warning", "Method may be missing 'end' keyword")
        }
        
    case "class":
        // Check for proper class naming (CamelCase)
        if name := p.GetNodeName(node, source); name != "" {
            if !p.isValidClassName(name) {
                p.addDiagnostic(diagnostics, node, "warning", "Class name should be CamelCase")
            }
        }
    }
    
    // Recursively check children
    for i := 0; i < int(node.ChildCount()); i++ {
        p.checkRubySyntaxPatterns(node.Child(i), source, diagnostics)
    }
}
```

### 5. Plugin Versioning and Compatibility

For external plugins, implement versioning:

```go
// Plugin interface with versioning
type VersionedProvider interface {
    provider.LanguageProvider
    Version() string
    MinMorfxVersion() string
    MaxMorfxVersion() string
}

// Plugin implementation
func (p *RubyProvider) Version() string { return "1.2.3" }
func (p *RubyProvider) MinMorfxVersion() string { return "2.0.0" }
func (p *RubyProvider) MaxMorfxVersion() string { return "2.9.9" }

// Registry checks compatibility
func (r *Registry) LoadPlugin(path string) error {
    // ... load plugin
    
    if vp, ok := provider.(VersionedProvider); ok {
        if !r.isCompatible(vp) {
            return fmt.Errorf("plugin version %s incompatible with morfx %s",
                vp.Version(), r.morfxVersion)
        }
    }
    
    return r.RegisterProvider(provider)
}
```

---

This comprehensive guide provides everything you need to create a robust language provider for morfx. Remember that the key to a great provider is understanding both the universal concepts that morfx uses and the specific AST structure of your target language.

For additional examples, study the existing providers in [`internal/lang/`](../../internal/lang/) and don't hesitate to contribute improvements back to the project!