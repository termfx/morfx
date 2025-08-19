# DSL Reference

The morfx Domain-Specific Language (DSL) provides a unified query syntax that works across all programming languages. This reference documents the complete syntax, operators, node kinds, and patterns supported by morfx.

## Table of Contents

- [Overview](#overview)
- [Basic Query Syntax](#basic-query-syntax)
- [Node Kinds](#node-kinds)
- [Operators](#operators)
- [Pattern Matching](#pattern-matching)
- [Hierarchical Queries](#hierarchical-queries)
- [Logical Operations](#logical-operations)
- [Attributes and Constraints](#attributes-and-constraints)
- [Language Examples](#language-examples)
- [Common Query Patterns](#common-query-patterns)
- [Advanced Patterns](#advanced-patterns)
- [Best Practices](#best-practices)

## Overview

The morfx DSL is designed to be:
- **Language-agnostic**: Same syntax works across Go, Python, JavaScript, TypeScript, etc.
- **Intuitive**: Uses familiar programming terms from different languages
- **Efficient**: Optimized for CLI usage with single-character primary operators
- **Flexible**: Supports aliases for different operator styles

## Basic Query Syntax

### Simple Query Format

```
kind:pattern [attributes...]
```

- **`kind`**: The type of code construct to find
- **`pattern`**: Name pattern to match (supports wildcards)
- **`attributes`**: Optional additional constraints

### Examples

```bash
# Find all functions named "test"
function:test

# Find all variables starting with "user"
variable:user*

# Find all classes ending with "Service"
class:*Service

# Find functions with type constraints
function:calculate string
```

## Node Kinds

The DSL supports universal node kinds that map to language-specific constructs. Each kind accepts multiple aliases for flexibility.

### Core Language Constructs

#### Functions and Methods
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `function` | `func`, `def`, `fn`, `sub`, `procedure` | Functions, methods, procedures |
| `method` | `method` | Class/object methods |

**Examples:**
```bash
function:calculate    # Universal term
func:Calculate       # Go style
def:calculate        # Python style
fn:calculate         # Rust style
```

#### Variables and Constants
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `variable` | `var`, `let`, `variable` | Variable declarations |
| `constant` | `const`, `final`, `readonly`, `immutable` | Constant declarations |

**Examples:**
```bash
variable:userName     # Universal term
var:userName         # Go/JavaScript style
let:userName         # JavaScript style
const:API_KEY        # Constant declaration
```

#### Classes and Types
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `class` | `class`, `struct`, `type` | Classes, structs, types |
| `interface` | `interface`, `protocol`, `trait` | Interfaces, protocols, traits |
| `enum` | `enum`, `enumeration` | Enumerations |
| `type` | `type` | Type definitions, aliases |

**Examples:**
```bash
class:User           # Universal term
struct:User          # Go style
type:User            # TypeScript/Go style
interface:Readable   # Interface definition
```

#### Imports and Dependencies
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `import` | `import`, `require`, `include`, `use`, `using`, `from` | Import statements |

**Examples:**
```bash
import:fmt           # Universal term
require:lodash       # Node.js style
use:std             # Rust style
include:header       # C/C++ style
```

### Code Structure

#### Fields and Properties
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `field` | `field`, `property`, `attribute`, `member`, `slot` | Struct fields, class properties |

#### Function Calls
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `call` | `call`, `invoke`, `apply`, `execute` | Function/method calls |

#### Assignments
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `assignment` | `assignment`, `assign`, `set` | Variable assignments |

### Control Flow

#### Conditions
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `condition` | `if`, `switch`, `case`, `when`, `match`, `condition` | Conditional statements |

#### Loops
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `loop` | `for`, `while`, `do`, `foreach`, `repeat`, `loop` | Loop constructs |

#### Exception Handling
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `try_catch` | `try`, `catch`, `except`, `rescue`, `finally` | Exception handling |
| `return` | `return`, `yield` | Return statements |
| `throw` | `throw`, `raise`, `panic` | Exception throwing |

### Documentation and Metadata

#### Comments
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `comment` | `comment`, `doc`, `documentation` | Code comments |

#### Decorators
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `decorator` | `decorator`, `annotation` | Decorators, annotations |

#### Parameters
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `parameter` | `parameter`, `param`, `argument`, `arg` | Function parameters |

#### Blocks and Scopes
| Universal Kind | Aliases | Maps To |
|----------------|---------|---------|
| `block` | `block`, `scope`, `begin`, `end` | Code blocks |

## Operators

morfx supports multiple operator syntaxes for maximum flexibility. Single characters are primary for CLI efficiency, with familiar aliases available.

### Logical Operators

#### AND Operator
| Primary | Aliases | Description |
|---------|---------|-------------|
| `&` | `&&`, `and`, `AND` | Both conditions must be true |

**Examples:**
```bash
# Primary (most efficient)
function:test* & class:User

# Aliases (also supported)
function:test* && class:User
function:test* and class:User
```

#### OR Operator
| Primary | Aliases | Description |
|---------|---------|-------------|
| `|` | `||`, `or`, `OR` | Either condition can be true |

**Examples:**
```bash
# Primary
function:test* | function:spec*

# Aliases
function:test* || function:spec*
function:test* or function:spec*
```

#### NOT Operator
| Primary | Aliases | Description |
|---------|---------|-------------|
| `!` | `not`, `NOT` | Negates the condition |

**Examples:**
```bash
# Primary
!function:deprecated*

# Aliases
not function:deprecated*
NOT function:deprecated*
```

### Hierarchical Operator

#### Parent-Child Relationship
| Operator | Description | Example |
|----------|-------------|---------|
| `>` | Parent contains child | `class:User > method:getName` |

**Usage:**
```bash
# Find methods inside specific classes
class:Controller > method:handle*

# Find variables inside specific functions
function:init > variable:config*

# Find calls inside specific methods
class:Database > method:connect > call:*
```

## Pattern Matching

### Wildcard Patterns

morfx supports standard wildcard patterns for flexible matching:

| Pattern | Description | Example | Matches |
|---------|-------------|---------|---------|
| `*` | Zero or more characters | `test*` | `test`, `testCase`, `testing` |
| `?` | Single character | `get?` | `getX`, `getY`, but not `get` |
| `exact` | Exact match | `main` | Only `main` |

### Pattern Examples

```bash
# Prefix matching
function:test*        # testCase, testMethod, testing

# Suffix matching  
function:*Test        # unitTest, integrationTest

# Contains matching
function:*test*       # containsTestData, testingModule

# Single character wildcards
variable:temp?        # temp1, tempX, but not temp10

# Exact matching
function:main         # Only "main"
```

## Hierarchical Queries

Hierarchical queries find child constructs within parent contexts using the `>` operator.

### Basic Hierarchy

```bash
# Methods within classes
class:User > method:getName

# Variables within functions
function:init > variable:*

# Calls within methods
method:process > call:validate
```

### Multi-Level Hierarchy

```bash
# Deep nesting: calls within methods within classes
class:Service > method:process > call:validate

# Mixed constructs
class:Controller > method:handle* > variable:request
```

### Complex Hierarchical Patterns

```bash
# Multiple children of same parent
class:User > (method:getName | method:setName)

# Negated hierarchical queries
class:User > !method:deprecated*

# Hierarchical with attributes
class:User > method:* public
```

## Logical Operations

### Combining Multiple Conditions

#### AND Operations
```bash
# Both conditions must match
function:test* & class:User
function:calculate & parameter:number

# Multiple AND operations
function:test* & !class:Mock & variable:data
```

#### OR Operations
```bash
# Either condition can match
function:test* | function:spec*
class:Service | class:Controller

# Multiple OR operations  
variable:user* | variable:account* | variable:profile*
```

#### Complex Boolean Logic
```bash
# Parentheses for precedence (conceptual - not implemented)
(function:test* | function:spec*) & class:User

# Negated OR
!(function:deprecated* | function:legacy*)

# Mixed operations
function:test* & (class:User | class:Account) & !variable:temp*
```

## Attributes and Constraints

### Type Constraints

Add type information as additional constraints:

```bash
# Variables with specific types
variable:user string
variable:count int
variable:items []string

# Functions with return types
function:calculate int
function:getName string
```

### Multiple Constraints

```bash
# Multiple attributes
function:process string error
variable:config map[string]interface{}
class:User public
```

### Language-Specific Attributes

Different languages support different attributes:

#### Go Attributes
```bash
function:Calculate public    # Exported function
variable:config private      # Unexported variable
struct:User                  # Go struct
```

#### Python Attributes
```bash
def:calculate @staticmethod  # Static method
class:User public           # Public class
variable:_private           # Private variable
```

#### JavaScript/TypeScript Attributes
```bash
function:calculate async     # Async function
variable:user const         # Const variable
class:User export          # Exported class
```

## Language Examples

### Go Examples

```bash
# Go functions
func:Calculate
func:main
func:init

# Go structs and types
struct:User
type:Config
interface:Reader

# Go packages and imports
import:fmt
import:encoding/json

# Go variables and constants
var:config
const:DefaultTimeout
```

### Python Examples

```bash
# Python functions and methods
def:calculate
def:__init__
def:process_data

# Python classes
class:User
class:DataProcessor

# Python imports
import:json
from:datetime
require:requests

# Python variables
variable:user_name
variable:data_list
```

### JavaScript/TypeScript Examples

```bash
# JavaScript functions
function:calculate
function:processData
fn:handleClick

# JavaScript classes and objects
class:User
class:Component

# JavaScript imports
import:lodash
require:fs
import:React

# JavaScript variables
variable:userName
let:config
const:API_KEY
```

### Universal Examples (Work in All Languages)

```bash
# Universal terms work everywhere
function:test*           # Functions in any language
class:User              # Classes/structs in any language
variable:config         # Variables in any language
import:*                # Imports in any language
```

## Common Query Patterns

### Testing Patterns

```bash
# Find all test functions
function:test*
function:*Test
def:test_*

# Find test classes
class:*Test
class:Test*
class:*Spec

# Find mock objects
class:Mock*
variable:mock*
function:createMock*

# Exclude test code
function:* & !function:test*
class:* & !class:*Test
```

### Configuration Patterns

```bash
# Find configuration
variable:config*
variable:*Config
const:*_CONFIG

# Find environment variables
variable:ENV_*
variable:*_ENV
const:NODE_ENV
```

### API and Service Patterns

```bash
# Find API endpoints
method:get*
method:post*
method:handle*
function:*Handler

# Find service classes
class:*Service
class:*Provider
class:*Client

# Find database operations
method:find*
method:save*
method:delete*
function:query*
```

### Error Handling Patterns

```bash
# Find error handling
try:*
catch:*
function:handle*Error
variable:*Error

# Find validation
function:validate*
function:*Valid
method:isValid*
```

## Advanced Patterns

### Refactoring Patterns

```bash
# Find deprecated code
function:deprecated*
class:*Deprecated
comment:*deprecated*

# Find TODO comments
comment:*TODO*
comment:*FIXME*
comment:*HACK*

# Find unused code (combine with usage analysis)
function:* & !call:*
class:* & !class:*
```

### Architecture Patterns

```bash
# Find controllers
class:*Controller
class:Controller*

# Find models
class:*Model
class:Model*
struct:*Model

# Find utilities
function:util*
class:*Util
class:Util*

# Find factories
function:create*
function:new*
method:factory*
```

### Security Patterns

```bash
# Find authentication code
function:auth*
function:*Auth
method:authenticate*
class:*Auth*

# Find password handling
variable:*password*
variable:*Password*
function:hash*
function:encrypt*

# Find SQL queries (potential injection points)
variable:*query*
variable:*Query*
function:exec*
method:query*
```

## Best Practices

### 1. Use Specific Patterns When Possible

```bash
# Good: Specific
function:calculateTax

# Less good: Too broad
function:calculate*
```

### 2. Combine Operators Effectively

```bash
# Good: Clear intent
function:test* & class:User & !method:deprecated*

# Less good: Overly complex
function:* & (class:* | struct:*) & !variable:* & method:*
```

### 3. Use Language-Appropriate Terms

```bash
# Go: Use Go terms
func:Calculate & struct:User

# Python: Use Python terms  
def:calculate & class:User

# Universal: Works everywhere
function:calculate & class:User
```

### 4. Leverage Hierarchical Queries

```bash
# Good: Context-aware
class:UserController > method:authenticate

# Less good: Context-free
method:authenticate
```

### 5. Use Negation Wisely

```bash
# Good: Exclude specific unwanted matches
function:* & !function:test* & !function:deprecated*

# Less good: Overly broad exclusion
function:* & !function:*
```

### 6. Choose Efficient Operators

```bash
# Most efficient (single character)
function:test* & class:User

# Also supported (but more typing)
function:test* and class:User
function:test* && class:User
```

---

The morfx DSL is designed to be powerful yet intuitive, allowing you to express complex code queries in a natural way that works across all programming languages. As you become familiar with the syntax, you'll discover that the same conceptual queries work whether you're working with Go, Python, JavaScript, or any other supported language.

For more examples and advanced usage patterns, see the [Creating Providers Guide](../guides/CREATING_PROVIDERS.md) and [Plugin Architecture Documentation](../architecture/PLUGIN_ARCHITECTURE.md).