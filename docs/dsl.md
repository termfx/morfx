# Morfx Structural DSL

Morfx accepts a compact structural DSL anywhere a tool accepts a query:

- use `dsl` with read tools such as `query` and `file_query`;
- use `target_dsl` with mutation tools such as `replace`, `delete`,
  `insert_before`, `insert_after`, `append`, file mutation tools, and recipe
  steps.

The DSL compiles to `core.AgentQuery`. It does not bypass confidence scoring,
dry-run checks, staged apply, provider validation, or recipe safety gates.

## Quick Examples

```txt
func:Handle*
!func:Test*
func:* > call:os.Getenv
class:UserController >> method:index
(func:* | method:*) > call:fetch
call:$client.$method
call:fetch arg0="/api/user"
import:* source=react
func:init before=func:run
struct:* > field:Secret string
field:Secret type=string visibility=private
class:* > method:render
def:* > call:os.getenv
return:*
```

## Grammar

```txt
expression  = or
or          = and ("|" and)*
and         = contains ("&" contains)*
contains    = unary ((">" | ">>") unary)*
unary       = "!" unary | primary
primary     = selector | "(" expression ")"
selector    = kind ":" pattern attributes*
attributes  = shorthand_type | key "=" value
```

Operator precedence, from strongest to weakest:

1. `!` negation
2. `>` / `>>` contains descendant or direct semantic child
3. `&` intersection
4. `|` union

Use parentheses when an agent or human might read the expression two ways.

```txt
(func:* | method:*) > call:fetch
func:* > (call:os.Getenv | call:viper.GetString)
```

`>` means the left selector contains a descendant matching the right selector.
It is not limited to direct children.

```txt
func:* > call:os.Getenv
class:* > method:render > call:setState
```

`>>` means the child selector must be a direct semantic child of the left
selector. Morfx treats common wrapper nodes such as class bodies and statement
blocks as transparent, so this stays useful across tree-sitter grammars.

```txt
class:UserController >> method:index
func:load >> return:*
```

When the left side is a compound expression, the child selector is distributed
over the operands. For example:

```txt
(func:* | method:*) > call:fetch
```

is equivalent to:

```txt
(func:* > call:fetch) | (method:* > call:fetch)
```

## Selector Shape

Every selector starts with:

```txt
kind:pattern
```

In short examples, read this as `kind:name`: the provider-specific selector
kind on the left, and the matched element name or wildcard pattern on the
right.

`kind` is provider-owned. Core parses it but does not decide what `func`,
`def`, `method`, `class`, or `field` means. Each language provider maps those
words to its own tree-sitter node types.

`pattern` uses shell-style wildcards through Go `path.Match`:

```txt
func:Handle*
class:*Controller
call:os.*
field:Secret
```

Pattern captures use `$name` inside the pattern. A capture matches that portion
of the selected name and returns it in the match `captures` object.

```txt
call:$callee
call:$client.$method
```

For `api.fetch`, the second selector returns:

```json
{"captures":{"client":"api","method":"fetch"}}
```

Use `*` for any name:

```txt
return:*
func:*
```

An empty pattern is treated as `*`, so `func:` is accepted as `func:*`.

## Attributes

Attributes further constrain a selector. The shorthand form remains supported:

```txt
field:Secret string
```

This is equivalent to:

```txt
field:Secret type=string
```

Use explicit key/value attributes for agent-generated queries:

```txt
field:Secret type=string
method:render visibility=public
call:fetch arg0="/api/user"
import:* source=react
func:init before=func:run
```

Current provider support is intentionally conservative. Go supports `type`
constraints for fields and declarations such as:

```txt
struct:* > field:Secret string
struct:* > field:Secret type=string
```

Unsupported attributes are ignored unless a provider implements validation for
them. Agents should prefer attributes documented for the target language.

Morfx also supports a small set of cross-provider predicate attributes:

| Attribute | Meaning |
| --- | --- |
| `text=<pattern>` | Match against the selected node source text |
| `source=<pattern>` | Match import/use/source selectors by extracted name or node text |
| `arg=<pattern>` | Match any call argument |
| `arg0=<pattern>`, `arg1=<pattern>` | Match a specific zero-based call argument |
| `before=<selector>` | Match when a sibling selector appears after this node |
| `after=<selector>` | Match when a sibling selector appears before this node |

Use quotes for argument or source values that contain punctuation:

```txt
call:fetch arg0="/api/user"
import:* source="react"
```

## Common Selectors

These selectors are intended to be broadly useful. Exact behavior is still
provider-owned because language grammars differ.

| Selector | Meaning |
| --- | --- |
| `function`, `func`, `fn` | Function-like declarations or expressions |
| `method` | Method declarations or method-like members |
| `class` | Class declarations or expressions |
| `struct` | Struct/type records where the language supports them |
| `interface`, `iface` | Interface declarations |
| `type` | Type aliases or type declarations |
| `field`, `property`, `prop` | Struct/class/interface fields or properties |
| `variable`, `var` | Variable declarations or assignments |
| `constant`, `const` | Constant declarations |
| `import`, `use`, `from` | Imports or namespace uses |
| `call` | Function/method call expressions |
| `assignment`, `assign` | Assignment expressions/statements |
| `return` | Return statements |
| `condition`, `if` | Conditional statements |
| `loop`, `for` | Loop statements |
| `block` | Block/body nodes |
| `comment`, `comments` | Comments |

## Provider Vocabulary

### Go

Common selectors:

```txt
func:Load*
method:ServeHTTP
struct:Config
interface:Reader
field:Secret type=string
call:os.Getenv
assign:fileHash
return:*
if:*
for:*
block:*
import:fmt
```

Go owns `func` and `fn`; it does not treat Python `def` as a function alias.

### JavaScript

Common selectors:

```txt
function:load*
func:load*
method:render
class:*Controller
field:state
property:state
var:cache
const:API_URL
let:ready
arrow:*
call:fetch
return:*
if:*
for:*
import:react
export:*
```

### TypeScript

TypeScript includes JavaScript selectors plus type-level selectors:

```txt
interface:User
type:UserId
enum:Status
member:Active
signature:*
property:name
call:fetch
return:*
```

### PHP

Common selectors:

```txt
function:getUser
method:index
class:UserController
trait:Auditable
interface:Repository
property:name
var:user
const:VERSION
namespace:App\\Http
use:App\\Models\\User
call:strtoupper
return:*
if:*
foreach:*
```

Use `method:* > call:foo` to find methods that call a function.

### Python

Common selectors:

```txt
def:load_user
function:load_user
class:User
var:cache
assign:cache
import:os
from:django.conf
decorator:cached_property
lambda:*
call:os.getenv
return:*
if:*
for:*
```

Python owns `def`; other providers should not interpret it unless they choose
to.

## Agent Usage Rules

Prefer DSL when the target is structural:

```json
{
  "language": "go",
  "path": "./config.go",
  "dsl": "func:* > call:os.Getenv"
}
```

Prefer JSON `query` when the agent already has a precise `AgentQuery` object or
needs programmatic composition.

For mutation tools, use `target_dsl`:

```json
{
  "language": "go",
  "path": "./config.go",
  "target_dsl": "func:LoadConfig > call:os.Getenv",
  "replacement": "func LoadConfig() Config { return Config{} }"
}
```

For recipes, use `target_dsl` inside each step:

```json
{
  "name": "remove-env-readers",
  "dry_run": true,
  "steps": [
    {
      "name": "delete env readers",
      "method": "delete",
      "scope": {
        "path": ".",
        "include": ["**/*.go"],
        "language": "go"
      },
      "target_dsl": "func:* > call:os.Getenv"
    }
  ]
}
```

## Good Queries

Find functions or methods that call an API:

```txt
(func:* | method:*) > call:fetch
```

Find Go structs with a secret string field:

```txt
struct:* > field:Secret type=string
```

Find Python functions that read environment variables:

```txt
def:* > call:os.getenv
```

Find non-test Go functions:

```txt
func:* & !func:Test*
```

Find PHP methods returning from a body:

```txt
method:* > return:*
```

Capture member calls:

```txt
call:$client.$method
```

Find calls by argument:

```txt
call:fetch arg0="/api/user"
```

Find direct class members:

```txt
class:* >> method:render
```

Find ordered siblings:

```txt
func:init before=func:run
```

## Limits

Captures are query-time bindings. They are returned with matches, but they are
not yet template variables for replacements. Do not generate replacement
templates such as:

```txt
replacement:"$client.$method()"
```

Use capture output to inspect matches first, then send explicit replacement
content.

The DSL does not yet support arbitrary boolean predicates, full typed captures,
or language-server-level symbol resolution. `>>` is direct semantic containment,
not a promise that the underlying tree-sitter node is an immediate raw child in
every grammar. Argument matching compares argument source text rather than
evaluating code. Use provider tests or JSON queries for behavior beyond the
documented DSL.
