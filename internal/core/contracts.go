// Package core contains pure language-agnostic data structures for Morfx.
// These contracts define the universal interface that ALL language providers must adhere to.
//
// IMPORTANT: This file contains ONLY pure data structures with NO methods.
// No language-specific dependencies are allowed here.
package core

// NodeKind represents universal AST node kinds that all languages must map to.
// These are the foundational building blocks of code structure across all programming languages.
type NodeKind string

const (
	// KindFunction represents function definitions, procedures, or methods
	KindFunction NodeKind = "function"

	// KindVariable represents variable declarations and definitions
	KindVariable NodeKind = "variable"

	// KindClass represents class, struct, or type definitions
	KindClass NodeKind = "class"

	// KindMethod represents methods within classes or structs
	KindMethod NodeKind = "method"

	// KindImport represents import statements, includes, or requires
	KindImport NodeKind = "import"

	// KindConstant represents constant declarations
	KindConstant NodeKind = "constant"

	// KindField represents struct fields, class properties, or object attributes
	KindField NodeKind = "field"

	// KindCall represents function calls, method invocations, or procedure calls
	KindCall NodeKind = "call"

	// KindAssignment represents variable assignments or mutations
	KindAssignment NodeKind = "assignment"

	// KindCondition represents if statements, switch cases, or conditional expressions
	KindCondition NodeKind = "condition"

	// KindLoop represents for loops, while loops, or iteration constructs
	KindLoop NodeKind = "loop"

	// KindBlock represents code blocks, scopes, or compound statements
	KindBlock NodeKind = "block"

	// KindComment represents code comments or documentation
	KindComment NodeKind = "comment"

	// KindDecorator represents decorators, annotations, or attributes
	KindDecorator NodeKind = "decorator"

	// KindType represents type definitions, aliases, or type annotations
	KindType NodeKind = "type"

	// KindInterface represents interface definitions or contracts
	KindInterface NodeKind = "interface"

	// KindEnum represents enumeration definitions
	KindEnum NodeKind = "enum"

	// KindParameter represents function or method parameters
	KindParameter NodeKind = "parameter"

	// KindReturn represents return statements
	KindReturn NodeKind = "return"

	// KindThrow represents throw or raise statements
	KindThrow NodeKind = "throw"

	// KindTryCatch represents try-catch or exception handling blocks
	KindTryCatch NodeKind = "try_catch"
)

// ScopeType defines the hierarchical scopes where operations can be applied.
// These represent universal code organization concepts across all languages.
type ScopeType string

const (
	// ScopeFile represents file-level scope (global scope within a file)
	ScopeFile ScopeType = "file"

	// ScopeClass represents class-level scope (within a class or struct definition)
	ScopeClass ScopeType = "class"

	// ScopeFunction represents function-level scope (within a function or method body)
	ScopeFunction ScopeType = "function"

	// ScopeBlock represents block-level scope (within a code block or compound statement)
	ScopeBlock ScopeType = "block"

	// ScopeNamespace represents namespace or module-level scope
	ScopeNamespace ScopeType = "namespace"

	// ScopePackage represents package-level scope
	ScopePackage ScopeType = "package"
)

// Query represents a parsed DSL query in a completely language-agnostic way.
// This structure contains only the essential elements needed to describe
// what to search for, without any language-specific implementation details.
type Query struct {
	// Kind specifies the universal node kind to match
	Kind NodeKind

	// Pattern contains the name/identifier pattern to match (e.g., function name, variable name)
	Pattern string

	// Attributes contains additional matching criteria as key-value pairs
	// Examples: {"type": "string", "visibility": "public", "static": "true"}
	Attributes map[string]string

	// Operator defines how to combine this query with others ("&&", "||", ">", "!")
	Operator string

	// Children contains nested queries for complex matching scenarios
	Children []Query

	// Scope specifies where this query should be applied
	Scope ScopeType

	// Raw preserves the original DSL string for debugging and error reporting
	Raw string
}

// Location represents a position within source code in a language-agnostic way.
// This structure uses only basic file positioning without any language-specific constructs.
type Location struct {
	// File is the path to the source file
	File string

	// StartLine is the starting line number (1-based)
	StartLine int

	// EndLine is the ending line number (1-based)
	EndLine int

	// StartCol is the starting column number (1-based)
	StartCol int

	// EndCol is the ending column number (1-based)
	EndCol int

	// StartByte is the starting byte offset in the file
	StartByte int

	// EndByte is the ending byte offset in the file
	EndByte int
}

// Result represents a language-agnostic match result from a query evaluation.
// This structure contains NO language-specific dependencies and can represent
// any code construct from any programming language.
type Result struct {
	// Kind is the universal node kind that was matched
	Kind NodeKind

	// Name is the identifier or name of the matched construct
	Name string

	// Location specifies where the match was found in the source code
	Location Location

	// Content is the raw text content of the matched construct
	Content string

	// Metadata contains additional language-agnostic information about the match
	// Examples: {"type": "string", "visibility": "public", "parameters": "2"}
	Metadata map[string]any

	// ParentKind is the kind of the immediate parent node (for context)
	ParentKind NodeKind

	// ParentName is the name of the immediate parent construct (for context)
	ParentName string

	// Scope indicates the scope type where this result was found
	Scope ScopeType
}

// ResultSet represents a collection of query results.
// This is a pure data structure with no methods - result manipulation
// should be handled by separate utility functions.
type ResultSet struct {
	// Results contains all the matched results
	Results []*Result

	// QueryHash is a hash of the original query for caching purposes
	QueryHash string

	// TotalMatches is the total number of matches found
	TotalMatches int

	// ExecutionTimeMs is the time taken to execute the query in milliseconds
	ExecutionTimeMs int64
}

// NodeMapping defines how universal node kinds map to language-specific AST nodes.
// This structure bridges the gap between universal concepts and language implementations.
type NodeMapping struct {
	// Kind is the universal node kind
	Kind NodeKind

	// NodeTypes contains the language-specific AST node type names
	NodeTypes []string

	// NameCapture specifies how to extract the name/identifier from the AST node
	NameCapture string

	// TypeCapture specifies how to extract type information from the AST node
	TypeCapture string

	// Template is a tree-sitter query template for matching this construct
	Template string

	// Attributes defines additional language-specific attributes that can be captured
	Attributes map[string]string

	// Priority indicates the matching priority (higher numbers = higher priority)
	Priority int
}

// QueryOptions contains configuration options for query execution.
// These options control how queries are processed and evaluated.
type QueryOptions struct {
	// CaseSensitive determines if pattern matching should be case-sensitive
	CaseSensitive bool

	// UseRegex enables regular expression matching for patterns
	UseRegex bool

	// MaxDepth limits the maximum depth of nested queries
	MaxDepth int

	// Timeout sets the maximum execution time in milliseconds
	Timeout int64

	// IncludeContent determines if result content should be populated
	IncludeContent bool

	// IncludeMetadata determines if result metadata should be populated
	IncludeMetadata bool
}

// ValidationError represents errors that occur during query or result validation.
type ValidationError struct {
	// Code is a machine-readable error code
	Code string

	// Message is a human-readable error message
	Message string

	// Field indicates which field caused the validation error
	Field string

	// Value is the invalid value that caused the error
	Value any
}

// ProviderCapabilities describes what features a language provider supports.
// This helps the core engine understand what operations are available.
type ProviderCapabilities struct {
	// SupportedKinds lists all the NodeKind values this provider can handle
	SupportedKinds []NodeKind

	// SupportedScopes lists all the ScopeType values this provider supports
	SupportedScopes []ScopeType

	// SupportsRegex indicates if the provider supports regex pattern matching
	SupportsRegex bool

	// SupportsNesting indicates if the provider supports nested queries
	SupportsNesting bool

	// MaxQueryDepth is the maximum nesting depth supported
	MaxQueryDepth int

	// SupportsValidation indicates if the provider can validate code snippets
	SupportsValidation bool

	// SupportsFormatting indicates if the provider can format code
	SupportsFormatting bool

	// SupportsImportOrganization indicates if the provider can organize imports
	SupportsImportOrganization bool
}

// QuickCheckDiagnostic represents a syntax or semantic issue found during quick validation.
// This is a pure language-agnostic structure for reporting code issues.
type QuickCheckDiagnostic struct {
	// Severity indicates the diagnostic level ("error", "warning", "info")
	Severity string

	// Message is the human-readable diagnostic message
	Message string

	// Line is the line number (1-based)
	Line int

	// Column is the column number (1-based)
	Column int

	// Code is an optional diagnostic code
	Code string
}

// FuzzyProvider defines the minimal interface needed by the fuzzy resolver.
// This avoids circular imports by providing only what the fuzzy matching needs.
type FuzzyProvider interface {
	// TranslateQuery converts a DSL query to tree-sitter query
	TranslateQuery(q *Query) (string, error)

	// GetSitterLanguage returns the tree-sitter language for parsing
	// Note: This returns interface{} to avoid importing tree-sitter in contracts
	GetSitterLanguage() any
}

// PipelineProvider defines the interface needed by the core pipeline.
// This avoids circular imports by providing only what the pipeline needs.
type PipelineProvider interface {
	// Lang returns the language identifier
	Lang() string

	// TranslateQuery converts a DSL query to tree-sitter query
	TranslateQuery(q *Query) (string, error)

	// GetSitterLanguage returns the tree-sitter language for parsing
	// Note: This returns interface{} to avoid importing tree-sitter in contracts
	GetSitterLanguage() any

	// AppendPoint finds insertion point for AppendToBody operation
	// Note: anchor is interface{} to avoid importing tree-sitter in contracts
	AppendPoint(anchor any, source []byte) (int, error)

	// ValidateSnippet validates code snippet
	// Note: context is interface{} to avoid importing tree-sitter in contracts
	ValidateSnippet(snippet string, context any, source []byte) error

	// OrganizeImports organizes imports in source code
	OrganizeImports(source []byte) ([]byte, error)

	// Format formats source code
	Format(source []byte) ([]byte, error)

	// QuickCheck performs quick validation
	QuickCheck(source []byte) []QuickCheckDiagnostic
}
