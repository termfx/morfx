# morfx

[![Build Status](https://github.com/termfx/morfx/workflows/CI/badge.svg)](https://github.com/termfx/morfx/actions)
[![Coverage](https://img.shields.io/badge/coverage-85%25-green.svg)](https://github.com/termfx/morfx)
[![Go Report Card](https://goreportcard.com/badge/github.com/termfx/morfx)](https://goreportcard.com/report/github.com/termfx/morfx)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)

> Code surgery at scale. Transform and refactor multi-language codebases with deterministic, AST-based precision.

## ✨ What is `morfx`?

`morfx` is a code transformation engine that understands your code, not just your text. Instead of relying on fragile regular expressions or the non-deterministic "magic" of an LLM, `morfx` parses your code into an **Abstract Syntax Tree (AST)** and allows you to operate on it with surgical precision.

It's not `sed` on steroids. It's **Terraform for your codebase**. It allows you to declare transformations, preview the impact, and apply them with the confidence that you're modifying the code's true structure, not just its textual representation.

## 🎯 The Problem (The Pain)

Large-scale refactoring is hell. We've all been there, stuck in a cycle of hope-driven development:

1. **The Brittle Regex:** You spend hours crafting the "perfect" regex, a masterpiece of lookaheads and capture groups. It works on your test cases. You run it on the codebase, cross your fingers, and it inevitably breaks something subtle in a file you didn't think to check. It doesn't understand scope, context, or comments—it only understands text.
2. **The Unmaintainable Script:** You write a one-off Python script for a critical refactor. It's a tangled mess of string manipulation and file I/O, but it gets the job done. It works once. Six months later, it's a piece of technical debt in your tools repository. Nobody knows how it works, nobody dares to touch it, and it's easier to rewrite it from scratch than to fix it.
3. **The LLM Black Box:** You ask an AI to "refactor all foo functions to return an error". Sometimes, it's brilliant, saving you hours. Other times, it hallucinates, subtly changes logic, or gets stuck in a loop, leaving you with a mess that takes even longer to clean up. It's not repeatable, it's not auditable, and it's certainly not built for production-critical transformations.

> We were tired of tools that guess, hope, or hallucinate. We wanted a tool that knows.

## 🧠 The `morfx` Philosophy

`morfx` is built on a set of core beliefs that directly address the pain of code transformation.

- **💎 Determinism > Probability:** This is our core promise. The same transformation, on the same code, must always produce the exact same result. There are no surprises, no random variations, no hallucinations. This is the confidence to run a critical refactor on a Friday afternoon.
- **🌳 Structure > Text:** We operate on the Abstract Syntax Tree, the true, semantic structure of your code. `morfx` understands the difference between a variable named `myVar` and a string literal that says "myVar". It knows what a function is, what its parameters are, and what it returns. It doesn't get confused by comments or formatting because it sees the code the way your compiler does.
- **🛡️ Safety by Default:** Fear is the enemy of good refactoring. `morfx` provides a safety net that encourages bold changes. Every operation is logged, auditable, and reversible. With a built-in staging area and a mechanical rollback system, you can experiment without the fear of leaving your codebase in a broken state.
- **🧱 Extreme Composability:** Real-world refactoring is rarely a single operation. A powerful Domain-Specific Language (DSL) and a Provider architecture allow you to chain simple, atomic transformations into complex, automated "playbooks." These playbooks can perform large-scale migrations, enforce coding standards, or upgrade dependencies across multiple languages with a single command.

## 🤔 When should you use `morfx`?

Choosing the right tool is critical. Here's where `morfx` fits in your arsenal:

| Tool                   | Use Case                                                                                                            |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `sed` / `grep` / `awk` | You need to replace simple, unambiguous text in a file. Quick and dirty.                                            |
| Scripts (Python/JS)    | You need a one-off refactor for a specific, well-defined problem and don't care about long-term maintenance.        |
| LLM / AI Agent         | You want to generate new code or perform an exploratory refactor. You accept variability.                           |
| `morfx`                | **You need to apply a complex code transformation, at scale, in a way that is safe, repeatable, and auditable. ✨** |

## 🔥 Key Features

- **🌐 Multi-Language Engine:** A Provider architecture supporting Go, JavaScript, Python, and TypeScript, trivially extensible to other languages.
- **🔍 Surgical Precision DSL:** Forget regex. Use a Domain-Specific Language designed for navigating the AST (e.g., `"function:GetUser && !import:legacy_db"`).
- **📊 Transactional Operations:** Every run is logged to a local SQLite database. You get a complete audit trail and the ability to rollback any operation.
- **🛡️ Staging & Atomicity:** Operations are first prepared in a staging area (`.morfx/`). Changes are only committed to disk if everything is correct, ensuring you never leave your code in a broken state.
- **⚡ Systems-Level Performance:** Written in Go to be ridiculously fast and concurrent, capable of processing multi-million-line codebases.
- **🧪 Golden Snapshot Testing:** Built-in support for snapshot testing to validate your transformations against "golden" versions.
- **🔧 Extensibility:** Easily extend `morfx` with new languages, providers, and transformations.

## 🏗️ Plugin Architecture

morfx achieves true language agnosticism through a sophisticated plugin architecture where the core engine operates on universal abstractions, completely decoupled from specific programming languages.

### Universal DSL Across Languages

The same DSL syntax works seamlessly across all supported languages. Write once, run everywhere:

```bash
# Python developers using familiar terms
morfx "def:test* & class:User" *.py

# Go developers using Go vocabulary
morfx "func:Test* & struct:User" *.go

# JavaScript developers with their style
morfx "function:test* & class:User" *.js

# Universal syntax works everywhere
morfx "function:test* & class:User" *.{py,go,js,ts}
```

### Zero-Coupling Design

- **Core Engine**: Knows nothing about specific languages
- **Universal Parser**: Accepts all common programming terms (`func`/`def`/`function`)
- **Language Providers**: Handle language-specific AST translation
- **Dynamic Loading**: Add new languages via plugins without recompilation

### Efficient Operator Design

Optimized for CLI efficiency with single-character primary operators:

```bash
# Primary operators (most efficient)
morfx "func:Test* & !struct:mock"        # & | ! >

# Familiar aliases also supported
morfx "func:Test* && not struct:mock"    # && || not
morfx "func:Test* and not struct:mock"   # and or not
```

### Cross-Language Query Examples

The plugin architecture enables powerful cross-language operations:

```bash
# Find all test functions across languages
morfx "function:test*" src/

# Complex queries work identically everywhere
morfx "(def:parse* | func:compile*) & !import:deprecated" .

# Hierarchical queries understand language structure
morfx "class:Controller > method:handle*" src/
```

### Extensibility

Adding new language support is straightforward:

- **Built-in Languages**: Go, Python, JavaScript, TypeScript
- **Plugin Support**: Load additional languages from `.so` files
- **Provider Interface**: Implement one interface, get full integration
- **Community Plugins**: Extend to any language with a Tree-sitter grammar

For detailed information about the architecture and creating custom providers, see:

- **[Plugin Architecture Guide](docs/architecture/PLUGIN_ARCHITECTURE.md)** - Deep dive into the zero-coupling design
- **[Creating Providers Guide](docs/guides/CREATING_PROVIDERS.md)** - Step-by-step provider development
- **[DSL Reference](docs/api/DSL_REFERENCE.md)** - Complete syntax and examples

## 🚀 Quick Start

Let's say you want to rename every instance of a variable `oldVar` to `newVar` inside a specific function, `doSomething`.

### `main.go` - Before:

```go
package main

import "fmt"

func doSomething() {
    oldVar := "hello"
    fmt.Println(oldVar)
}

func doSomethingElse() {
    oldVar := "goodbye" // We don't want to touch this one
    fmt.Println(oldVar)
}
```

### The Command:

```
morfx --target ./... --query "function:doSomething > variable:oldVar" --operation replace --replacement "newVar" --commit
```

### `main.go` - After:

```go
package main

import "fmt"

func doSomething() {
    newVar := "hello" // <-- Renamed!
    fmt.Println(newVar) // <-- Renamed!
}

func doSomethingElse() {
    oldVar := "goodbye" // <-- Untouched, as intended.
    fmt.Println(oldVar)
}
```

`morfx` understood the scope and only changed the variable inside the `doSomething` function, something a simple text search-and-replace could never do safely because it operates on structure, not just text.

---

## 📦 Installation

### From Source

Ensure you have Go 1.24+ installed.

```bash
# Clone the repository
git clone https://github.com/termfx/morfx.git
cd morfx

# Build the binary
make build

# (Optional) Move the binary to your PATH
mv bin/morfx /usr/local/bin/
```

### From Pre-compiled Binaries

Pre-compiled binaries for Linux, macOS, and Windows are available on the GitHub Releases page.

## 📖 Usage

`morfx` operates via command-line flags. The core flags are:

| Flag          | Description                                                          | Example                       |
| ------------- | -------------------------------------------------------------------- | ----------------------------- |
| --target      | The file or directory to process. Supports glob patterns.            | `--target ./...`              |
| --query       | The DSL query to find target code nodes.                             | `--query "func:Test*"`        |
| --operation   | The action to perform (replace, insert-before, etc.).                | `--operation delete`          |
| --replacement | The new content for replace or insert operations.                    | `--replacement "new content"` |
| --commit      | Applies the changes to disk. Without it, morfx runs in dry-run mode. | `--commit`                    |
| --lang        | Manually specify the language provider.                              | `--lang go`                   |
| --json        | Output results in a machine-readable JSON format.                    | `--json`                      |

### Common Commands

Find all functions named `Legacy` in a directory (dry-run):

```bash
morfx --target ./internal/ --query "func:Legacy"
```

Delete all console.log calls in JavaScript files and apply the changes:

```bash
morfx --target "\*_/_.js" --query "call:console.log" --operation delete --commit
```

Insert a comment before every function in a file:

```bash
morfx --target main.go --query "func:\*" --operation insert-before --replacement "// TODO: Refactor this" --commit
```

## 📚 Documentation

For more in-depth information, please see the [full documentation](docs/README.md).

[Architecture](docs/architecture/README.md) - A deep dive into the internal design of morfx.

[Contributing](docs/contributing/README.md) - How to get involved with the project.

[DSL Guide](docs/guides/README.md) - A comprehensive guide to the morfx query language.

[Deployment](docs/deployment/README.md) - Instructions for deploying morfx in different environments.

## 🔒 Security

`morfx` takes security seriously. Please review our [Security Policy](SECURITY.md) and report any vulnerabilities responsibly.

## 📄 License

This project is licensed under the [MIT License](LICENSE).

`morfx` - Transform code with precision and confidence. 🎯
