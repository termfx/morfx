# Testing Documentation

This directory contains comprehensive testing documentation for morfx.

## Contents

- [Testing Strategy](./testing-strategy.md) - Overall testing approach and philosophy
- [Coverage Guidelines](./COVERAGE.md) - Coverage requirements and monitoring
- [Test Categories](./test-categories.md) - Unit, integration, and e2e test organization
- [Performance Testing](./performance-testing.md) - Benchmarking and performance regression detection
- [Test Data Management](./test-data.md) - Managing test fixtures and golden snapshots

## Testing Overview

morfx employs a comprehensive testing strategy with multiple layers:

### Test Categories
- **Unit Tests**: Fast, isolated tests for individual components
- **Integration Tests**: Component interaction and database integration
- **End-to-End Tests**: Full workflow validation with real codebases
- **Golden Tests**: Snapshot testing for DSL query results
- **Performance Tests**: Benchmarks and regression detection

### Coverage Requirements
- **Minimum Coverage**: 80% across all packages
- **Critical Path Coverage**: 95% for core transformation logic
- **New Code Coverage**: 100% for new features and bug fixes

### Test Organization
```
internal/
├── pkg/
│   ├── component.go
│   ├── component_test.go      # Unit tests
│   └── integration_test.go    # Integration tests
└── lang/golang/
    ├── provider_test.go       # Unit tests
    ├── integration_test.go    # Language-specific integration
    ├── e2e_test.go           # End-to-end validation
    └── testdata/             # Test fixtures and snapshots
```

### Running Tests
```bash
# All tests with coverage
make test

# Specific test categories
make test-race          # Race condition detection
make test-verbose       # Detailed output
make test-one PKG=./internal/db  # Single package

# Performance and validation
make gate              # Critical validation suite
make regen-snapshots   # Update golden snapshots
```

### Coverage Monitoring
- Local coverage reports: `make coverage-html`
- CI coverage enforcement: 80% minimum threshold
- Coverage trends tracked in pull requests
- Detailed coverage analysis available in HTML reports