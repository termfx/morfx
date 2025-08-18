# Architecture Documentation

This directory contains comprehensive system architecture documentation for morfx.

## Contents

- **System Overview** - High-level architecture and component relationships
- **Database Schema** - SQLite database design and migration patterns
- **Language Providers** - Multi-language support architecture
- **Tree-sitter Integration** - Code parsing and AST manipulation
- **Configuration System** - Configuration management and validation
- **Security Architecture** - Encryption and security patterns

## Key Architectural Decisions

Morfx follows a modular, extensible architecture that supports:

1. **Multi-language Support** - Pluggable language providers (Go, JavaScript, Python, TypeScript)
2. **Database-driven Operations** - SQLite with FTS5 for efficient code indexing and search
3. **Tree-sitter Parser Integration** - Language-agnostic code parsing and manipulation
4. **Encryption Support** - Optional data encryption for sensitive transformations
5. **CLI-first Design** - Command-line interface with rich configuration options

## Getting Started

1. Review the [System Overview](./system-overview.md) for architectural context
2. Understand the [Language Provider System](./language-providers.md)
3. Explore [Database Schema](./database-schema.md) for persistence patterns