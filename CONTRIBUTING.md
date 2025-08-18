# Contributing to morfx

Thank you for your interest in contributing to morfx! This document provides guidelines and information for contributors to help ensure a smooth collaboration process.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Contributing Process](#contributing-process)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Review Process](#review-process)
- [Community and Communication](#community-and-communication)

## Code of Conduct

This project follows the [Go Community Code of Conduct](https://golang.org/conduct). By participating, you agree to uphold this code. Please report unacceptable behavior to [conduct@morfx.dev](mailto:conduct@morfx.dev).

## Getting Started

### Prerequisites

- **Go 1.24+**: The latest version of Go
- **Git**: For version control
- **Make**: For build automation
- **SQLite3**: For database functionality (with FTS5 support)
- **A Unix-like environment**: Linux, macOS, or WSL on Windows

### Quick Setup

```bash
# Fork and clone the repository
git clone https://github.com/termfx/morfx.git
cd morfx

# Install dependencies
go mod download

# Verify the setup works
make test

# Run the full quality gate
make quality-gate
```

### First-time Setup

1. **Install pre-commit hooks** (recommended):
   ```bash
   pip install pre-commit
   pre-commit install
   ```

2. **Configure Git** (if not already done):
   ```bash
   git config --global user.name "Your Name"
   git config --global user.email "your.email@example.com"
   ```

3. **Set up your development environment**:
   ```bash
   # Install additional Go tools
   go install golang.org/x/tools/cmd/goimports@latest
   go install mvdan.cc/gofumpt@latest
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   ```

## Development Environment

### Recommended Tools

- **Editor**: VS Code with Go extension, GoLand, or Vim with vim-go
- **Terminal**: Any Unix-compatible terminal
- **Database Tools**: SQLite browser for database inspection
- **Profiling**: Go's built-in profiling tools (`go tool pprof`)

### Environment Variables

morfx uses several environment variables for configuration:

```bash
export MORFX_ENCRYPTION_MODE=off        # or "blob" for encryption
export MORFX_MASTER_KEY=your_key_here   # for encryption mode
export MORFX_FORCE_NO_FTS5=1           # disable FTS5 for testing
```

### Project Structure

```
morfx/
├── cmd/                    # Main applications
│   ├── morfx/             # CLI application
│   └── morfx-provider-gen/ # Code generation tool
├── internal/              # Private packages
│   ├── cli/               # CLI logic
│   ├── config/            # Configuration management
│   ├── db/                # Database operations
│   ├── lang/              # Language providers
│   └── ...
├── docs/                  # Documentation
├── testdata/              # Test data and fixtures
└── scripts/               # Build and deployment scripts
```

## Contributing Process

### 1. Choose What to Contribute

Great first contributions include:

- **Documentation improvements**: Fix typos, clarify instructions, add examples
- **Bug fixes**: Address issues in the GitHub issue tracker
- **Test coverage**: Add tests for uncovered code
- **Language support**: Extend or improve language providers
- **Performance improvements**: Optimize hot paths
- **Feature implementations**: Implement requested features

### 2. Find or Create an Issue

Before starting work:

- **Check existing issues**: Look for related work or discussions
- **Create a new issue**: For bugs, features, or significant changes
- **Discuss your approach**: Get feedback before implementing large changes

### 3. Plan Your Contribution

For significant changes:

- **Design document**: Create an RFC for major features
- **Break down work**: Split large changes into smaller, reviewable pieces
- **Consider compatibility**: Ensure backward compatibility when possible

## Code Standards

morfx follows strict coding standards to ensure maintainability and consistency. Please review the [Code Standards](docs/contributing/CODE_STANDARDS.md) document for detailed guidelines.

### Key Standards Summary

- **Go conventions**: Follow effective Go and standard Go practices
- **Error handling**: Always handle errors explicitly with proper context
- **Testing**: Write comprehensive tests with 80%+ coverage
- **Documentation**: Document all public APIs and complex logic
- **Performance**: Consider performance implications, especially in hot paths

### Code Formatting

All code must be properly formatted:

```bash
# Format your code before committing
make fix

# Or run individual tools
go fmt ./...
goimports -w .
gofumpt -w .
```

## Testing Guidelines

### Test Requirements

- **Minimum 80% coverage**: All contributions must maintain coverage
- **Test categories**: Unit, integration, and end-to-end tests as appropriate
- **Edge cases**: Include error conditions and boundary testing
- **Race conditions**: Use `make test-race` to detect concurrency issues

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Check coverage
make coverage-check

# Run integration tests
make gate

# Test a specific package
make test-one PKG=./internal/parser
```

### Writing Good Tests

Follow these patterns:

```go
func TestParser_ParseQuery(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Query
        wantErr bool
    }{
        {
            name:    "valid query",
            input:   "func main",
            want:    &Query{Type: "func", Name: "main"},
            wantErr: false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewParser()
            got, err := parser.ParseQuery(tt.input)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseQuery() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseQuery() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Documentation

### Documentation Requirements

- **Public APIs**: All exported functions and types must have documentation
- **Complex logic**: Internal complexity should be explained with comments
- **Examples**: Include usage examples in documentation
- **Architecture**: Update architecture docs for significant changes

### Documentation Types

1. **Code comments**: For functions, types, and complex logic
2. **README updates**: For user-facing changes
3. **API documentation**: In the `docs/api/` directory  
4. **User guides**: In the `docs/guides/` directory
5. **Architecture docs**: In the `docs/architecture/` directory

### Documentation Style

```go
// ParseQuery parses a DSL query string and returns an AST representation.
// It supports nested expressions, operators, and function calls.
//
// Example:
//   query := "func main() { call fmt.Println }"
//   ast, err := ParseQuery(query)
//   if err != nil {
//       // handle error
//   }
//
// Returns an error if the query syntax is invalid.
func ParseQuery(query string) (*AST, error) {
    // implementation
}
```

## Submitting Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-python-support`
- `fix/parser-crash-on-empty-input`
- `docs/improve-getting-started-guide`
- `perf/optimize-database-queries`

### Commit Messages

Write clear, concise commit messages:

```
type(scope): short description

Longer explanation if needed, including:
- What changed and why
- Any breaking changes
- References to issues or RFCs

Fixes #123
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`

### Pull Request Process

1. **Create a fork**: Fork the repository to your GitHub account
2. **Create a branch**: Make changes in a feature branch
3. **Make changes**: Implement your changes following the guidelines
4. **Test locally**: Run `make quality-gate` to verify everything works
5. **Commit changes**: Use clear commit messages
6. **Push branch**: Push your branch to your fork
7. **Open PR**: Create a pull request with a clear description

### Pull Request Template

Your PR should include:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature  
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally (`make test`)
- [ ] Coverage maintained (`make coverage-check`)
- [ ] Integration tests pass (`make gate`)

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added for new functionality
```

## Review Process

### What to Expect

- **Initial feedback**: Within 2-3 business days
- **Review iterations**: May require multiple rounds of feedback
- **CI checks**: All automated checks must pass
- **Approvals**: At least one maintainer approval required

### Review Criteria

Reviewers will check:

- **Correctness**: Does the code work as intended?
- **Style**: Does it follow project conventions?
- **Tests**: Are there adequate tests?
- **Performance**: Are there performance implications?
- **Documentation**: Is it properly documented?
- **Compatibility**: Does it maintain backward compatibility?

### Addressing Review Comments

- **Be responsive**: Address feedback promptly
- **Ask questions**: If feedback is unclear, ask for clarification
- **Explain decisions**: Provide context for your implementation choices
- **Update documentation**: Keep docs in sync with code changes

## Community and Communication

### Getting Help

- **GitHub Discussions**: For questions and general discussion
- **GitHub Issues**: For bugs and feature requests
- **Code Review**: For implementation feedback
- **Documentation**: Check the `docs/` directory first

### Communication Guidelines

- **Be respectful**: Treat all community members with respect
- **Be constructive**: Provide actionable feedback
- **Be patient**: Maintainers and contributors are often volunteers
- **Be inclusive**: Welcome newcomers and different perspectives

### Recognition

Contributors are recognized through:

- **Contributor list**: Maintained in the repository
- **Release notes**: Significant contributions are highlighted
- **Issue/PR mentions**: Credit given in relevant discussions

## Development Tips

### Debugging

```bash
# Debug with verbose output
go test -v ./internal/parser -run TestSpecificFunction

# Run with race detection
go test -race ./...

# Profile performance
make profile

# Generate CPU profile
go test -cpuprofile=cpu.prof -bench=. ./internal/lang/golang/
go tool pprof cpu.prof
```

### Performance Testing

```bash
# Run benchmarks
make benchmark

# Compare performance
go test -bench=. -count=5 ./internal/parser > before.txt
# Make changes
go test -bench=. -count=5 ./internal/parser > after.txt
benchcmp before.txt after.txt
```

### Database Development

```bash
# Run tests with different database configurations
MORFX_ENCRYPTION_MODE=off make test
MORFX_FORCE_NO_FTS5=1 make test

# Inspect test databases
sqlite3 .morfx/test.db ".tables"
```

## License

By contributing to morfx, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).

---

Thank you for contributing to morfx! Your efforts help make code transformation more accessible and powerful for developers everywhere. 🚀