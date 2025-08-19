# API Reference Documentation

This directory contains comprehensive API documentation for morfx internal packages and interfaces.

## Contents

- **Core APIs** - Primary interfaces and contracts
- **Language Provider APIs** - Plugin interfaces for language support
- **Database APIs** - Data persistence and query interfaces  
- **Configuration APIs** - Configuration management interfaces
- **CLI APIs** - Command-line interface and runner APIs

## API Categories

### Core Components
- [`internal/model`](./core/model.md) - Core data models and types
- [`internal/registry`](./core/registry.md) - Component registry and dependency injection
- [`internal/config`](./core/config.md) - Configuration system
- [`internal/cli`](./core/cli.md) - CLI runner and dispatcher

### Language Support
- [`internal/provider`](./language/provider.md) - Provider contract interface
- [`internal/lang/golang`](./language/golang.md) - Go language provider
- [`internal/lang/javascript`](./language/javascript.md) - JavaScript provider
- [`internal/lang/python`](./language/python.md) - Python provider
- [`internal/lang/typescript`](./language/typescript.md) - TypeScript provider

### Data & Storage
- [`internal/db`](./data/database.md) - SQLite database operations
- [`internal/scanner`](./data/scanner.md) - File system scanning
- [`internal/writer`](./data/writer.md) - Output writing and staging

### Processing & Manipulation
- [`internal/parser`](./processing/parser.md) - Universal parsing interface
- [`internal/evaluator`](./processing/evaluator.md) - Query evaluation engine
- [`internal/manipulator`](./processing/manipulator.md) - Code manipulation operations
- [`internal/matcher`](./processing/matcher.md) - Pattern matching engine

## Usage Guidelines

- All public APIs must be documented with examples
- Breaking changes require version bumps and migration guides
- New APIs should include comprehensive test coverage
- Performance characteristics should be documented for critical paths