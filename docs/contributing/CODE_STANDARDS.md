# Code Standards and Best Practices

This document outlines the coding standards, conventions, and best practices for the morfx project. These standards ensure consistency, maintainability, and high code quality across the codebase.

## Table of Contents

- [Go Coding Standards](#go-coding-standards)
- [Naming Conventions](#naming-conventions)
- [Error Handling](#error-handling)
- [Testing Requirements](#testing-requirements)
- [Performance Guidelines](#performance-guidelines)
- [Code Review Checklist](#code-review-checklist)
- [Documentation Standards](#documentation-standards)

## Go Coding Standards

### Project Structure

Follow the standard Go project layout:

```
├── cmd/                    # Main applications
│   ├── morfx/             # CLI application
│   └── morfx-provider-gen/ # Code generation tool
├── internal/              # Private application code
│   ├── cli/               # CLI logic
│   ├── config/            # Configuration management
│   ├── db/                # Database operations
│   ├── lang/              # Language providers
│   └── ...
├── pkg/                   # Public library code (if any)
├── docs/                  # Documentation
├── scripts/               # Build and deployment scripts
└── testdata/              # Test data files
```

### Package Organization

- **One package per directory**: Each directory should contain exactly one package
- **Clear package names**: Package names should be short, lowercase, and descriptive
- **Avoid package name stuttering**: Don't repeat package name in type names

**Good:**

```go
package db

type Connection struct {} // Not DatabaseConnection
```

**Bad:**

```go
package database

type DatabaseConnection struct {}
```

### Code Formatting

- Use `gofmt` and `gofumpt` for consistent formatting
- Use `goimports` for import organization
- Use `gci` for import grouping (stdlib, external, internal)
- Line length: 100 characters maximum (enforced by `golines`)

### Import Organization

Imports should be grouped in this order:

1. Standard library packages
2. External dependencies
3. Internal packages (prefixed with module name)

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/spf13/pflag"

    "github.com/termfx/morfx/internal/config"
    "github.com/termfx/morfx/internal/model"
)
```

## Naming Conventions

### Variables and Functions

- **camelCase** for unexported names: `configPath`, `parseQuery`
- **PascalCase** for exported names: `ConfigPath`, `ParseQuery`
- **Descriptive names**: Avoid abbreviations unless they're well-known
- **Context variables**: Always name context variables `ctx`

```go
// Good
func BuildConfiguration(ctx context.Context, configPath string) (*Config, error) {
    // ...
}

// Bad
func BuildConfig(c context.Context, cp string) (*Cfg, error) {
    // ...
}
```

### Types and Interfaces

- **Nouns for types**: `User`, `Configuration`, `DatabaseConnection`
- **Verbs for interfaces**: `Reader`, `Writer`, `Parser`, `Evaluator`
- **Interface naming**: Single-method interfaces should end with `-er`

```go
type Parser interface {
    Parse(content []byte) (*AST, error)
}

type QueryEvaluator interface {
    Evaluate(ctx context.Context, query string) ([]Result, error)
}
```

### Constants and Variables

- **ALL_CAPS** for exported constants: `DEFAULT_TIMEOUT`
- **camelCase** for unexported constants: `defaultTimeout`
- **Descriptive names** for package-level variables

```go
const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

var (
    configMutex sync.RWMutex
    globalRegistry *Registry
)
```

## Error Handling

### Error Creation and Wrapping

- **Always handle errors explicitly** - never ignore errors silently
- **Wrap errors with context** using `fmt.Errorf` with `%w` verb
- **Return early** on errors to reduce nesting

```go
// Good
func ProcessFile(path string) error {
    file, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("failed to open file %q: %w", path, err)
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        return fmt.Errorf("failed to read file %q: %w", path, err)
    }

    if err := process(data); err != nil {
        return fmt.Errorf("failed to process file %q: %w", path, err)
    }

    return nil
}
```

### Custom Error Types

Use custom error types for errors that need special handling:

```go
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Is(target error) bool {
    _, ok := target.(*ValidationError)
    return ok
}
```

### Error Messages

- **Start with lowercase** unless beginning with proper noun
- **Be specific** about what failed and why
- **Include context** like file paths, IDs, or operation details
- **Avoid redundant information** that's already in the call stack

```go
// Good
return fmt.Errorf("failed to parse configuration file %q: invalid YAML syntax at line %d", path, lineNum)

// Bad
return fmt.Errorf("Error: Failed to parse file")
```

## Testing Requirements

### Test Organization

- **Table-driven tests** for multiple test cases
- **Separate test files** for each package: `package_test.go`
- **Test data** in `testdata/` directories
- **Integration tests** in separate files: `integration_test.go`

### Test Naming

```go
func TestFunction(t *testing.T) {}           // Unit test
func TestFunction_EdgeCase(t *testing.T) {}  // Specific scenario
func BenchmarkFunction(b *testing.B) {}      // Benchmark test
```

### Test Structure

Use the Arrange-Act-Assert pattern:

```go
func TestConfigParser_ParseFile(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        want     *Config
        wantErr  bool
    }{
        {
            name:    "valid configuration",
            input:   "testdata/valid-config.yaml",
            want:    &Config{Database: "sqlite://test.db"},
            wantErr: false,
        },
        {
            name:    "invalid yaml",
            input:   "testdata/invalid-config.yaml",
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            parser := NewConfigParser()

            // Act
            got, err := parser.ParseFile(tt.input)

            // Assert
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseFile() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseFile() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Coverage Requirements

- **Minimum 80% overall coverage** across all packages
- **95% coverage for critical paths** (core transformation logic)
- **100% coverage for new features** and bug fixes
- **Test edge cases** including error conditions

### Test Categories

1. **Unit Tests**: Fast, isolated tests for individual functions
2. **Integration Tests**: Test component interactions
3. **End-to-End Tests**: Test complete workflows
4. **Golden Tests**: Snapshot testing for DSL queries
5. **Benchmark Tests**: Performance regression detection

## Performance Guidelines

### General Principles

- **Profile before optimizing** - use `go test -bench` and `go tool pprof`
- **Measure actual performance** impact of changes
- **Optimize hot paths** identified through profiling
- **Consider memory allocations** in frequently called code

### Memory Management

- **Minimize allocations** in hot paths
- **Use sync.Pool** for frequently allocated objects
- **Prefer slices over maps** when order matters and keys are sequential
- **Pre-allocate slices** when size is known: `make([]T, 0, expectedSize)`

```go
// Good - pre-allocate with known capacity
results := make([]Result, 0, len(input))
for _, item := range input {
    if processed := process(item); processed != nil {
        results = append(results, *processed)
    }
}

// Bad - repeated allocations
var results []Result
for _, item := range input {
    if processed := process(item); processed != nil {
        results = append(results, *processed)
    }
}
```

### Concurrency

- **Use channels** for goroutine communication
- **Always handle goroutine lifecycle** - avoid goroutine leaks
- **Use context.Context** for cancellation and timeouts
- **Synchronize access** to shared state with mutexes or channels

```go
func ProcessConcurrently(ctx context.Context, items []Item) <-chan Result {
    results := make(chan Result, len(items))

    go func() {
        defer close(results)

        var wg sync.WaitGroup
        for _, item := range items {
            wg.Add(1)
            go func(item Item) {
                defer wg.Done()

                select {
                case <-ctx.Done():
                    return
                case results <- processItem(item):
                }
            }(item)
        }
        wg.Wait()
    }()

    return results
}
```

### Database Performance

- **Use transactions** for multiple operations
- **Prepare statements** for repeated queries
- **Use appropriate indexes** for query performance
- **Batch operations** when possible

## Code Review Checklist

### Functionality

- [ ] Code correctly implements the intended functionality
- [ ] Edge cases are handled appropriately
- [ ] Error conditions are properly managed
- [ ] No obvious bugs or logic errors

### Design and Architecture

- [ ] Code follows SOLID principles
- [ ] Appropriate separation of concerns
- [ ] Consistent with existing architecture patterns
- [ ] No unnecessary complexity or over-engineering

### Go Standards Compliance

- [ ] Follows Go idioms and conventions
- [ ] Proper error handling with context wrapping
- [ ] Appropriate use of interfaces and composition
- [ ] Concurrency primitives used correctly

### Testing

- [ ] Adequate test coverage (minimum 80%)
- [ ] Tests cover edge cases and error conditions
- [ ] Benchmarks for performance-critical code
- [ ] Golden tests updated if DSL changes

### Performance

- [ ] No obvious performance bottlenecks
- [ ] Memory allocations minimized in hot paths
- [ ] Goroutines have proper lifecycle management
- [ ] Database queries are efficient

### Documentation

- [ ] Public APIs are documented
- [ ] Complex logic has explanatory comments
- [ ] README updated if user-facing changes
- [ ] Architecture docs updated if needed

### Security

- [ ] No security vulnerabilities introduced
- [ ] Input validation for external data
- [ ] Sensitive data handled appropriately
- [ ] No secrets in code or logs

## Documentation Standards

### Code Comments

- **Public APIs must have comments** starting with the function/type name
- **Complex algorithms** should have explanatory comments
- **TODO comments** should include issue numbers or context

```go
// ParseQuery parses a DSL query string and returns an AST.
// It supports nested expressions, operators, and function calls.
// Returns an error if the query syntax is invalid.
func ParseQuery(query string) (*AST, error) {
    // Implementation uses recursive descent parsing
    // to handle nested expressions efficiently
    parser := newParser(query)
    return parser.parse()
}
```

### README Files

- Each package should have clear purpose documentation
- Include usage examples for complex packages
- Document any special setup or configuration requirements

### API Documentation

- Use Go doc conventions for generating documentation
- Include examples in doc comments when helpful
- Document error conditions and return values

---

Following these standards ensures that morfx maintains high code quality, performance, and maintainability. All contributors should familiarize themselves with these guidelines and use them during development and code review processes.
