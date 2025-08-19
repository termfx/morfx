# Plugin Architecture

The morfx plugin architecture enables true language-agnostic code transformation through a sophisticated provider system. This document explains the design principles, core components, and architectural patterns that make morfx universally applicable across programming languages.

## Table of Contents

- [Overview](#overview)
- [Core Principles](#core-principles)
- [Architecture Diagram](#architecture-diagram)
- [Component Breakdown](#component-breakdown)
- [Dependency Injection Pattern](#dependency-injection-pattern)
- [Provider Interface](#provider-interface)
- [Universal Abstractions](#universal-abstractions)
- [Plugin Loading System](#plugin-loading-system)
- [Benefits](#benefits)
- [Design Decisions](#design-decisions)

## Overview

morfx achieves language independence through a **zero-coupling architecture** where the core engine operates exclusively on universal abstractions, delegating all language-specific operations to pluggable providers. This design allows the same DSL queries and transformation logic to work across any programming language with an AST parser.

```
Universal DSL Query: "func:Test* & !struct:mock"
│
├─ Go Provider     → "(function_declaration name: (identifier) @name) (#match? @name "Test.*")"
├─ Python Provider → "(function_def name: (identifier) @name) (#match? @name "Test.*")"
├─ JS Provider     → "(function_declaration id: (identifier) @name) (#match? @name "Test.*")"
└─ Any Provider    → Language-specific Tree-sitter query
```

## Core Principles

### 1. Zero Language Knowledge in Core

The core modules ([`internal/core`](../../internal/core/), [`internal/parser`](../../internal/parser/), [`internal/evaluator`](../../internal/evaluator/), [`internal/registry`](../../internal/registry/)) contain **zero imports** of language-specific code. They operate entirely on universal contracts.

**Why this matters**: New languages can be added without modifying core components, and the core remains stable regardless of language-specific changes.

### 2. Universal Abstractions

All programming language constructs are mapped to universal [`NodeKind`](../../internal/core/contracts.go:13) constants:

```go
type NodeKind string

const (
    KindFunction   NodeKind = "function"   // Functions, methods, procedures
    KindVariable   NodeKind = "variable"   // Variables, let, const
    KindClass      NodeKind = "class"      // Classes, structs, types
    KindImport     NodeKind = "import"     // Import, require, use
    // ... 21 universal kinds total
)
```

**Why this matters**: The same conceptual query works across languages - "find all functions" means the same thing whether it's `func` in Go, `def` in Python, or `function` in JavaScript.

### 3. Provider-Based Extension

Language support comes from providers implementing the [`LanguageProvider`](../../internal/provider/contract.go:26) interface. Each provider translates between universal concepts and language-specific AST nodes.

**Why this matters**: Adding a new language requires only implementing one interface - no core modifications needed.

### 4. Dependency Injection

The universal evaluator receives providers as dependencies, allowing the same evaluation logic to work with any language:

```go
type UniversalEvaluator struct {
    provider provider.LanguageProvider  // Injected at runtime
}
```

**Why this matters**: Single evaluator implementation for all languages, easy testing with mock providers, clean separation of concerns.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   cmd/morfx │────│ File Auto-  │────│   Registry  │     │
│  │             │    │ Detection   │    │   Lookup    │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                      Core Engine                            │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │  Universal  │────│  Universal  │────│  Universal  │     │
│  │    Parser   │    │  Evaluator  │    │ Manipulator │     │
│  │             │    │  (Injected) │    │             │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                  Provider Interface                         │
│              ┌─────────────────────────────┐                │
│              │    LanguageProvider         │                │
│              │  ┌─────────────────────────┐│                │
│              │  │ TranslateQuery()        ││                │
│              │  │ GetNodeKind()           ││                │
│              │  │ GetNodeName()           ││                │
│              │  │ GetSitterLanguage()     ││                │
│              │  └─────────────────────────┘│                │
│              └─────────────────────────────┘                │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                Language Providers                           │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌─────────┐  │
│  │    Go     │  │  Python   │  │JavaScript │  │ Plugin  │  │
│  │ Provider  │  │ Provider  │  │ Provider  │  │ (.so)   │  │
│  │           │  │           │  │           │  │         │  │
│  └───────────┘  └───────────┘  └───────────┘  └─────────┘  │
└─────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    Tree-sitter Layer                        │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌─────────┐  │
│  │ go-tree-  │  │ python-   │  │javascript-│  │ custom- │  │
│  │ sitter    │  │ tree-     │  │ tree-     │  │ tree-   │  │
│  │           │  │ sitter    │  │ sitter    │  │ sitter  │  │
│  └───────────┘  └───────────┘  └───────────┘  └─────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Component Breakdown

### Core Contracts ([`internal/core/contracts.go`](../../internal/core/contracts.go))

Pure data structures with **no behavior** that define the universal interface:

- **[`NodeKind`](../../internal/core/contracts.go:10)**: Universal AST node types
- **[`Query`](../../internal/core/contracts.go:101)**: Parsed DSL query representation  
- **[`Result`](../../internal/core/contracts.go:153)**: Language-agnostic match results
- **[`ScopeType`](../../internal/core/contracts.go:77)**: Universal scope hierarchy
- **[`NodeMapping`](../../internal/core/contracts.go:200)**: Provider translation rules

### Universal Parser ([`internal/parser/universal.go`](../../internal/parser/universal.go))

Completely language-agnostic DSL parser that:

- Accepts **all common programming terms** from different languages
- Maps everything to universal [`NodeKind`](../../internal/core/contracts.go:10) constants
- Supports **multiple operator syntaxes** (`&`/`&&`/`and`, `|`/`||`/`or`)
- Produces only pure [`core.Query`](../../internal/core/contracts.go:101) structs

```go
// All of these produce the same universal Query:
"func:Test*"      // Go style
"def:Test*"       // Python style  
"function:Test*"  // JavaScript style
```

### Provider Interface ([`internal/provider/contract.go`](../../internal/provider/contract.go))

The **minimal but complete** interface that all language providers must implement:

```go
type LanguageProvider interface {
    // Metadata
    Lang() string
    Aliases() []string  
    Extensions() []string
    GetSitterLanguage() *sitter.Language
    
    // Core translation methods
    TranslateKind(kind core.NodeKind) []NodeMapping
    TranslateQuery(q *core.Query) (string, error)
    
    // AST introspection
    GetNodeKind(node *sitter.Node) core.NodeKind
    GetNodeName(node *sitter.Node, source []byte) string
    ParseAttributes(node *sitter.Node, source []byte) map[string]string
    
    // Scope analysis
    GetNodeScope(node *sitter.Node) core.ScopeType
    FindEnclosingScope(node *sitter.Node, scope core.ScopeType) *sitter.Node
    
    // Language services
    OrganizeImports(source []byte) ([]byte, error)
    Format(source []byte) ([]byte, error)
    QuickCheck(source []byte) []core.QuickCheckDiagnostic
}
```

### Registry System ([`internal/registry/registry.go`](../../internal/registry/registry.go))

Thread-safe provider management with:

- **Multi-key lookup**: By name, alias, or file extension
- **Dynamic plugin loading**: Load providers from `.so` files at runtime
- **Conflict resolution**: Prevents registration conflicts
- **Auto-discovery**: Scans standard plugin directories

```go
registry.GetProvider("go")          // By canonical name
registry.GetProvider("golang")      // By alias
registry.GetProvider(".go")         // By extension
registry.GetProviderForFile("main.go") // Auto-detect from filename
```

## Dependency Injection Pattern

The universal evaluator demonstrates the dependency injection pattern:

```go
// Single evaluator implementation for ALL languages
type UniversalEvaluator struct {
    provider provider.LanguageProvider  // Injected dependency
}

func (e *UniversalEvaluator) Evaluate(query *core.Query, source []byte) (*core.ResultSet, error) {
    // 1. Use provider to translate universal query to Tree-sitter query
    tsQuery, err := e.provider.TranslateQuery(query)
    if err != nil {
        return nil, err
    }
    
    // 2. Parse source using provider's Tree-sitter language
    parser := sitter.NewParser()
    parser.SetLanguage(e.provider.GetSitterLanguage())
    tree, err := parser.ParseCtx(context.Background(), nil, source)
    if err != nil {
        return nil, err
    }
    
    // 3. Execute Tree-sitter query
    q, err := sitter.NewQuery([]byte(tsQuery), e.provider.GetSitterLanguage())
    if err != nil {
        return nil, err
    }
    
    // 4. Convert matches back to universal results using provider
    cursor := sitter.NewQueryCursor()
    matches := cursor.QueryCaptures(q, tree.RootNode(), source)
    
    results := &core.ResultSet{}
    for match := range matches {
        for _, capture := range match.Captures {
            result := &core.Result{
                Kind: e.provider.GetNodeKind(capture.Node),
                Name: e.provider.GetNodeName(capture.Node, source),
                // ... universal result fields
            }
            results.Results = append(results.Results, result)
        }
    }
    
    return results, nil
}
```

**Benefits of this pattern:**
- **Single implementation**: Same evaluation logic for all languages
- **Easy testing**: Inject mock providers for unit tests
- **Runtime flexibility**: Switch providers based on file type
- **Clear separation**: Core logic separate from language specifics

## Provider Interface

### Translation Methods

Providers bridge universal concepts to language-specific implementations:

```go
// Go Provider example
func (p *GoProvider) TranslateKind(kind core.NodeKind) []NodeMapping {
    switch kind {
    case core.KindFunction:
        return []NodeMapping{
            {
                Kind:        core.KindFunction,
                NodeTypes:   []string{"function_declaration", "method_declaration"},
                NameCapture: "@name",
                Template:    `(function_declaration name: (identifier) @name %s)`,
            },
        }
    case core.KindClass:  // Go structs map to universal class concept
        return []NodeMapping{
            {
                Kind:        core.KindClass,
                NodeTypes:   []string{"type_declaration"},
                NameCapture: "@name", 
                Template:    `(type_declaration (type_spec name: (type_identifier) @name type: (struct_type) %s))`,
            },
        }
    }
}
```

### BaseProvider Helper

Providers can embed [`BaseProvider`](../../internal/provider/contract.go:117) for common functionality:

```go
type GoProvider struct {
    provider.BaseProvider  // Embedded for common methods
    dslVocabulary map[string]core.NodeKind  // Go-specific DSL terms
}
```

The [`BaseProvider`](../../internal/provider/contract.go:117) provides:
- **Query caching**: Avoid repeated translation work
- **Wildcard conversion**: Convert `*` and `?` to regex patterns  
- **Scope detection**: Default scope analysis algorithms
- **Query building**: Template-based Tree-sitter query construction

## Universal Abstractions

### NodeKind Hierarchy

All programming constructs map to 21 universal [`NodeKind`](../../internal/core/contracts.go:10) constants:

```
Core Language Constructs:
├─ KindFunction    (func, def, function, fn, sub, procedure)
├─ KindVariable    (var, let, variable)  
├─ KindConstant    (const, final, readonly, immutable)
├─ KindClass       (class, struct, type)
├─ KindMethod      (method - class-bound functions)
├─ KindInterface   (interface, protocol, trait)
├─ KindEnum        (enum, enumeration)
├─ KindField       (field, property, attribute, member)
├─ KindImport      (import, require, include, use, using)
└─ KindType        (type annotations, aliases)

Control Flow:
├─ KindCondition   (if, switch, case, when, match)
├─ KindLoop        (for, while, do, foreach, repeat)
├─ KindTryCatch    (try, catch, except, rescue, finally)
├─ KindReturn      (return, yield)
└─ KindThrow       (throw, raise, panic)

Operations:
├─ KindCall        (function calls, invocations)
├─ KindAssignment  (assignments, mutations)
└─ KindBlock       (code blocks, scopes)

Documentation:
├─ KindComment     (comments, documentation)
├─ KindDecorator   (decorators, annotations)
└─ KindParameter   (function/method parameters)
```

### Scope Hierarchy

Universal [`ScopeType`](../../internal/core/contracts.go:77) defines code organization:

```
ScopeFile      (global/module level)
├─ ScopePackage    (package/namespace level)
├─ ScopeClass      (class/struct level)  
│   ├─ ScopeFunction   (function/method level)
│   │   └─ ScopeBlock      (block level - if, for, etc.)
│   └─ ScopeFunction
└─ ScopeFunction   (top-level functions)
    └─ ScopeBlock
```

## Plugin Loading System

### Built-in Providers

Providers compiled into the binary are automatically registered:

```go
// cmd/morfx/providers.go
func init() {
    registry.RegisterProvider(golang.NewProvider())
    registry.RegisterProvider(python.NewProvider()) 
    registry.RegisterProvider(javascript.NewProvider())
    registry.RegisterProvider(typescript.NewProvider())
}
```

### External Plugins

Load language providers from `.so` files at runtime:

```go
// Auto-load from standard directories
registry.AutoRegister()

// Manual plugin loading  
registry.LoadPlugin("/path/to/ruby-provider.so")
registry.LoadPluginsFromDir("~/.morfx/plugins/")
```

### Plugin Structure

External plugins must export a `Provider` symbol:

```go
// ruby-provider plugin
package main

import "github.com/termfx/morfx/internal/provider"

var Provider provider.LanguageProvider = NewRubyProvider()

func NewRubyProvider() provider.LanguageProvider {
    // Implementation
}
```

## Benefits

### 1. True Language Agnosticism

- **Same DSL**: `"func:Test*"` works in Go, Python, JavaScript, etc.
- **Same evaluation logic**: Single evaluator for all languages
- **Same API**: Consistent interface regardless of language

### 2. Easy Extension

- **Zero core changes**: Add languages without modifying core
- **Standard interface**: Implement one interface, get full integration
- **Plugin support**: Load new languages at runtime

### 3. Testing & Maintenance  

- **Isolated testing**: Test core with mock providers
- **Language isolation**: Provider bugs don't affect core
- **Incremental updates**: Update language support independently

### 4. Performance

- **Shared infrastructure**: Tree-sitter parsing optimized once
- **Query caching**: Providers cache translated queries
- **Minimal overhead**: Direct interface calls, no reflection

### 5. Consistency

- **Universal concepts**: Same abstractions across languages
- **Predictable behavior**: Same query produces comparable results
- **Cross-language queries**: Mix languages in single transformation

## Design Decisions

### Why Single Characters for Primary Operators?

**Decision**: Use `&`, `|`, `!`, `>` as primary operators with aliases `&&`/`and`, `||`/`or`, `not`.

**Rationale**: 
- **CLI efficiency**: Fewer characters to type for common operations
- **Familiar alternatives**: Support double operators from C-family languages
- **English readability**: Support word operators for documentation

```bash
# Primary (most efficient)
morfx "func:Test* & !struct:mock" 

# Aliases (also supported)  
morfx "func:Test* && not struct:mock"
morfx "func:Test* and not struct:mock"
```

### Why Universal NodeKind Constants?

**Decision**: Map all language constructs to universal constants rather than language-specific strings.

**Rationale**:
- **Type safety**: Compile-time checking vs runtime string errors
- **Cross-language queries**: Same concept works everywhere
- **Tool integration**: IDEs can provide autocompletion
- **Extensibility**: Add new kinds without breaking existing code

### Why Provider Interface vs Abstract Classes?

**Decision**: Use Go interfaces rather than abstract base classes.

**Rationale**:
- **Go idioms**: Interfaces are the Go way of abstraction
- **Composition**: Embed [`BaseProvider`](../../internal/provider/contract.go:117) for shared behavior
- **Testing**: Easy to create mock implementations
- **Flexibility**: Providers can have different internal structures

### Why Tree-sitter vs Custom Parsers?

**Decision**: Use Tree-sitter for all language parsing.

**Rationale**:
- **Battle-tested**: Used by GitHub, Atom, Emacs, etc.
- **Performance**: Incremental parsing, error recovery
- **Language coverage**: 40+ languages already supported
- **Query language**: Powerful pattern matching built-in
- **Maintenance**: Don't maintain custom parsers

### Why Zero Core Dependencies?

**Decision**: Core modules import no language-specific code.

**Rationale**:
- **Stability**: Core remains stable regardless of language changes
- **Testing**: Test core independently of language implementations  
- **Plugin loading**: Load languages dynamically without compile-time knowledge
- **Clean boundaries**: Clear separation enables better architecture

---

This architecture enables morfx to be truly language-agnostic while maintaining high performance, type safety, and extensibility. The plugin system allows the tool to grow with new languages without requiring core modifications, making it a sustainable platform for code transformation across any programming ecosystem.