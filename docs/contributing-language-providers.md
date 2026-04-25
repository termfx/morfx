# Contributing Language Providers

Morfx language support is provider-based. A provider owns language-specific
syntax, AST node mapping, DSL aliases, name extraction, and append heuristics.
The shared base provider owns parser pooling, AST walking, matching, transforms,
confidence scoring, and file-level orchestration.

This guide documents the contract for adding a new built-in provider.

## Provider Contract

There are two layers:

- `providers.Provider` is the public runtime contract registered in
  `providers.Registry`.
- `providers/base.LanguageConfig` is the smaller contract most providers should
  implement, then wrap with `base.New(config)`.

Most new providers should follow the existing layout:

```txt
providers/<language>/
  provider.go
  config.go
  config_test.go
  provider_test.go
```

`provider.go` should register catalog metadata and expose `New()`:

```go
package rust

import (
    "github.com/oxhq/morfx/providers/base"
    "github.com/oxhq/morfx/providers/catalog"
)

func init() {
    catalog.Register(catalog.LanguageInfo{
        ID:         "rust",
        Extensions: (&Config{}).Extensions(),
    })
}

func New() *base.Provider {
    return base.New(&Config{})
}
```

`config.go` implements the language contract:

```go
type Config struct{}

func (c *Config) Language() string
func (c *Config) Extensions() []string
func (c *Config) GetLanguage() *sitter.Language
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string
func (c *Config) IsExported(name string) bool
func (c *Config) SupportedQueryTypes() []string
```

These methods are required by `providers/base.LanguageConfig`.

## Query Type Ownership

Providers own query vocabulary. Core parses the DSL shape, but it does not
decide that `func`, `def`, `function`, `fn`, `final`, or any other word means a
specific AST node. That decision belongs inside the selected language provider.

Implement `NormalizeQueryType` when a provider wants DSL aliases or public query
aliases to collapse to a canonical semantic type:

```go
func (c *Config) NormalizeQueryType(queryType string) string {
    switch strings.TrimSpace(queryType) {
    case "func", "fn":
        return "function"
    case "var":
        return "variable"
    default:
        return strings.TrimSpace(queryType)
    }
}
```

Then map canonical and accepted names to tree-sitter node types:

```go
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
    if nodes, ok := c.aliasMap()[queryType]; ok {
        return nodes
    }
    return []string{queryType}
}

func (c *Config) aliasMap() map[string][]string {
    return map[string][]string{
        "function": {"function_item", "closure_expression"},
        "class":    {"struct_item", "enum_item", "trait_item"},
        "call":     {"call_expression"},
    }
}
```

Keep `SupportedQueryTypes()` in sync with the keys exposed by `aliasMap()`.
That list is used by discovery surfaces such as MCP provider resources.

## Required Behavior

A provider must:

- return a stable language ID from `Language()`, such as `go`, `php`, or `rust`;
- return normalized file extensions from `Extensions()`, with leading dots;
- return the correct tree-sitter grammar from `GetLanguage()`;
- map semantic query types to concrete tree-sitter node types;
- extract names consistently for every mapped node type;
- return deterministic query results for the same source and query;
- support transforms through the shared base provider unless there is a clear
  reason to implement `providers.Provider` directly.

Name extraction is especially important. Query matching uses
`ExtractNodeName()`, wildcard matching, and the provider-local target metadata
created by the base provider. If a mapped node type can be anonymous, return an
empty string and let the base provider use `anonymous` where appropriate.

## Optional Hooks

The base provider detects optional interfaces on the config.

### `NormalizeQueryType`

Use this to keep language-specific DSL words in the provider. Python can map
`def` to `function`; Go should not.

### `ExpandMatches`

Use this when one tree-sitter node should produce multiple logical matches.
Examples include grouped imports, variable declarations, object fields, or
multi-binding declarations.

```go
func (c *Config) ExpandMatches(
    node *sitter.Node,
    source string,
    query core.AgentQuery,
) []base.Target
```

### `ValidateQueryAttributes`

Use this for provider-owned DSL/type constraints such as:

```txt
struct:* > field:Secret string
```

The parser stores `string` as an attribute. The provider decides how to inspect
the AST and compare that type constraint.

```go
func (c *Config) ValidateQueryAttributes(
    target base.Target,
    source string,
    attributes map[string]string,
) bool
```

Core already handles cross-provider attributes such as `text`, `source`,
`arg`, `arg0`, `before`, and `after`. Providers should reserve
`ValidateQueryAttributes` for language-specific constraints such as field
types, visibility, modifiers, or framework-specific metadata.

### `SmartAppend`

Use this when appending to the root or a target requires language-aware
placement, such as imports before declarations or methods inside a class body.

```go
func (c *Config) SmartAppend(
    source string,
    target *sitter.Node,
    content string,
) (modified string, handled bool)
```

Return `handled=false` to fall back to the base provider behavior.

### Node Validation Hooks

Some existing providers implement additional node validation methods used by the
base provider through optional interfaces. Use these only when node type alone
is too broad for the language grammar, for example distinguishing constructors,
type specs, properties, or semantic methods.

## Registration Checklist

1. Add the provider package under `providers/<language>`.
2. Add the tree-sitter grammar dependency to `go.mod`.
3. Implement `provider.go` and `config.go`.
4. Register the provider in `internal/runtime/registerBuiltInProviders`.
5. Add tests for metadata, query mapping, name extraction, query behavior,
   transform behavior, and malformed syntax.
6. Add DSL tests for provider-owned aliases, especially aliases that should not
   leak across languages.
7. Update README language support tables and any standalone examples if the new
   language should be public.

## Minimum Test Matrix

Every provider should have tests for:

- `Language()` and `Extensions()`;
- `MapQueryTypeToNodeTypes()` for every advertised semantic type;
- `SupportedQueryTypes()` including DSL aliases;
- `ExtractNodeName()` for each mapped declaration kind;
- `Query()` for functions/classes/types/variables/imports or equivalent core
  constructs;
- `Transform()` for replace/delete/insert/append where the language supports
  those operations;
- `Validate()` for valid and malformed source;
- DSL alias ownership, for example `def:*` working in Python but not in Go.

Before opening a provider PR, run:

```powershell
$env:CGO_ENABLED='1'
$env:CC='zig cc -target x86_64-windows-gnu'
go test ./providers/<language> ./providers/base ./providers ./...
```

Use the equivalent CGO-capable compiler setup on non-Windows platforms.

## Contributor Pitfalls

- Do not add global DSL translations in `core`; keep keyword meaning inside the
  provider.
- Do not expose an alias in `SupportedQueryTypes()` unless it maps and has test
  coverage.
- Do not rely on raw text matching for AST constructs that tree-sitter can
  expose structurally.
- Do not add broad node mappings without name extraction and transform tests.
- Do not claim a language is supported until query, transform, file-scope, MCP,
  and standalone paths can all reach it through the runtime registry.
