# Coverage Monitoring and Guidelines

This document outlines the coverage monitoring strategy, requirements, and best practices for the morfx project.

## Table of Contents

- [Coverage Requirements](#coverage-requirements)
- [Coverage Monitoring Tools](#coverage-monitoring-tools)
- [Local Development](#local-development)
- [CI/CD Integration](#cicd-integration)
- [Coverage Reports](#coverage-reports)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Coverage Requirements

### Minimum Thresholds

- **Overall Project Coverage**: 80% minimum
- **Critical Path Coverage**: 95% minimum for core transformation logic
- **New Code Coverage**: 100% for new features and bug fixes
- **Regression Coverage**: No decrease in existing coverage without justification

### Package-Level Requirements

Different packages have varying coverage requirements based on their criticality:

| Package Category                                                                 | Minimum Coverage | Rationale                             |
| -------------------------------------------------------------------------------- | ---------------- | ------------------------------------- |
| **Core Logic** (`internal/parser`, `internal/evaluator`, `internal/manipulator`) | 95%              | Critical transformation functionality |
| **Language Providers** (`internal/lang/*`)                                       | 90%              | Language-specific processing          |
| **Database** (`internal/db`)                                                     | 85%              | Data persistence and integrity        |
| **CLI** (`internal/cli`, `cmd/*`)                                                | 75%              | User interface and command handling   |
| **Utilities** (`internal/util`, `internal/types`)                                | 80%              | Supporting functionality              |

## Coverage Monitoring Tools

### Local Development Tools

#### 1. Make Targets

```bash
# Generate and view coverage report
make coverage

# Generate HTML coverage report and open in browser
make coverage-html

# Check coverage against threshold (80%)
make coverage-check

# Generate coverage badge information
make coverage-badge

# Generate all coverage reports for CI
make coverage-ci
```

#### 2. Go Tools Integration

```bash
# Manual coverage generation
go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...

# View coverage summary
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# View coverage for specific package
go tool cover -func=coverage.out | grep "internal/parser"
```

### CI/CD Integration

#### GitHub Actions

The CI pipeline automatically:

1. **Generates Coverage**: Runs tests with coverage on every push/PR
2. **Enforces Thresholds**: Fails builds if coverage drops below 80%
3. **Reports Results**: Uploads coverage to Codecov and generates artifacts
4. **Tracks Trends**: Monitors coverage changes over time

#### Coverage Artifacts

The CI generates several coverage artifacts:

- `coverage.out` - Machine-readable coverage data
- `coverage.html` - Human-readable HTML report
- `coverage.txt` - Text summary for quick review

## Coverage Reports

### Reading Coverage Reports

#### Command Line Output

```bash
$ make coverage

Coverage Analysis
==================
Coverage by Package:
-------------------
github.com/termfx/morfx/internal/parser        87.5%
github.com/termfx/morfx/internal/evaluator     92.3%
github.com/termfx/morfx/internal/db           85.2%
github.com/termfx/morfx/cmd/morfx             76.8%

Overall Coverage:
-----------------
Total: 85.4%
```

#### HTML Report Navigation

The HTML report (`coverage.html`) provides:

1. **Package Overview**: Coverage percentage for each package
2. **File-Level Details**: Line-by-line coverage visualization
3. **Uncovered Code**: Highlighted lines that need tests
4. **Function Coverage**: Per-function coverage statistics

### Understanding Coverage Colors

In HTML reports:

- 🟢 **Green**: Covered lines (executed by tests)
- 🔴 **Red**: Uncovered lines (not executed by tests)
- 🟡 **Yellow**: Partially covered lines (some branches not tested)

## Local Development

### Pre-commit Coverage Check

Coverage is automatically checked on commit via pre-commit hooks:

```bash
# Install pre-commit hooks
pre-commit install

# Run coverage check manually
pre-commit run go-test-coverage --all-files
```

### Development Workflow

1. **Write Tests First**: Follow TDD principles
2. **Check Coverage Locally**: Run `make coverage-check` before committing
3. **Review HTML Report**: Use `make coverage-html` to identify gaps
4. **Fix Coverage Issues**: Add tests for uncovered code
5. **Validate Quality**: Run `make quality-gate` for full validation

### Coverage-Driven Development

```bash
# Start with coverage check to see current state
make coverage-check

# Write your code and tests
# ...

# Check coverage again
make coverage-html

# Identify uncovered areas in the HTML report
# Add more tests

# Final validation
make quality-gate
```

## CI/CD Integration

### GitHub Actions Workflow

The CI workflow includes comprehensive coverage monitoring:

```yaml
# Coverage enforcement job
coverage:
  name: 📊 Coverage Analysis
  runs-on: ubuntu-latest
  needs: test
  steps:
    - name: Generate coverage report
      run: make test
    - name: Check coverage threshold
      run: make coverage-check
    - name: Upload coverage to Codecov
      uses: codecov/codecov-action@v3
```

### Codecov Integration

Coverage reports are automatically uploaded to Codecov, providing:

- **Trend Analysis**: Track coverage changes over time
- **Pull Request Comments**: Automatic coverage reports on PRs
- **Branch Comparison**: Compare coverage between branches
- **Sunburst Visualization**: Interactive coverage exploration

## Troubleshooting

### Common Coverage Issues

#### 1. Coverage Too Low

```bash
❌ Coverage 76.2% is below required 80%

Coverage by package:
  📉 github.com/termfx/morfx/cmd/morfx: 65.4%
  📉 github.com/termfx/morfx/internal/util: 72.1%
```

**Solutions**:

- Add tests for uncovered functions
- Remove dead code
- Focus on critical paths first

#### 2. Flaky Coverage Results

**Symptoms**: Coverage varies between runs
**Causes**: Race conditions, time-dependent tests, external dependencies
**Solutions**:

- Use `make test-race` to identify race conditions
- Mock external dependencies
- Use fixed time values in tests

#### 3. Coverage Reports Not Generated

**Symptoms**: `coverage.out` file missing
**Causes**: Test failures, build errors, incorrect flags
**Solutions**:

- Check test output for failures
- Verify build passes: `go build ./...`
- Run tests manually: `go test ./...`

### Debugging Coverage Issues

#### Identify Uncovered Code

```bash
# Find packages with low coverage
make coverage | grep -E "\b[0-7][0-9]\.[0-9]%"

# Get detailed function coverage
go tool cover -func=coverage.out | sort -k3 -n
```

#### Exclude Generated Code

Generated code can be excluded from coverage:

```go
//go:build ignore
// +build ignore

package main

// This file is excluded from coverage
```

Or use build tags in the coverage command:

```bash
go test -tags=!generated -cover ./...
```

## Best Practices

### Writing Testable Code

1. **Dependency Injection**: Make dependencies injectable for easier mocking
2. **Small Functions**: Break large functions into testable units
3. **Clear Interfaces**: Define clear contracts that are easy to mock
4. **Error Handling**: Test both success and error paths

### Effective Test Coverage

1. **Test Behavior, Not Implementation**: Focus on what the code does, not how
2. **Edge Cases**: Test boundary conditions and edge cases
3. **Error Scenarios**: Ensure all error paths are covered
4. **Integration Points**: Test component interactions

### Coverage vs. Quality

Remember: **100% coverage ≠ perfect tests**

Good coverage includes:

- ✅ All code paths (branches)
- ✅ Error conditions
- ✅ Edge cases
- ✅ Integration scenarios

Poor coverage includes:

- ❌ Only happy path tests
- ❌ Tests without assertions
- ❌ Mocking everything without integration tests

### Maintaining Coverage

1. **Monitor Trends**: Watch for coverage degradation over time
2. **Review PRs**: Ensure new code includes appropriate tests
3. **Refactor Safely**: Use coverage to verify refactoring doesn't break functionality
4. **Document Exceptions**: Clearly document and justify coverage exceptions

### Coverage Exclusions

Some code may legitimately be excluded from coverage:

- **Generated Code**: Auto-generated files
- **Main Functions**: Simple entry points
- **Build Tags**: Platform-specific code
- **Panic Handlers**: Error conditions that should never occur

Document all exclusions with clear rationale.

## Integration with Development Tools

### VS Code Integration

Add to your `.vscode/settings.json`:

```json
{
  "go.coverOnSave": true,
  "go.coverageDecorator": {
    "type": "gutter",
    "coveredHighlightColor": "rgba(64,128,64,0.5)",
    "uncoveredHighlightColor": "rgba(128,64,64,0.5)"
  }
}
```

### GoLand Integration

1. Go to **Run** > **Run with Coverage**
2. Set coverage threshold in settings
3. Enable coverage gutter marks

## Metrics and Monitoring

### Key Metrics to Track

1. **Overall Coverage Percentage**: Project-wide coverage
2. **Package Coverage Distribution**: Coverage by package
3. **Coverage Trend**: Changes over time
4. **New Code Coverage**: Coverage for recent changes
5. **Critical Path Coverage**: Coverage for essential functionality

### Setting Up Alerts

Configure alerts for:

- Coverage dropping below thresholds
- Large decreases in coverage (>5%)
- Critical packages falling below requirements

---

By following these guidelines and utilizing the available tools, morfx maintains high-quality test coverage that ensures reliability and maintainability of the codebase.
