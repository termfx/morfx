# QUERY_DSL.md

> Formal definition of MORFX's node selection DSL  
> Version: v0.1 — focused exclusively on Go using Tree-sitter

---

## 🔧 Purpose

This DSL enables **precise structural queries** to locate AST nodes for use in `get`, `delete`, `insert-before`, and similar commands.  
It is deterministic, case-sensitive, and designed for atomic code transformations.

---

## ✨ General Syntax

```txt
[node_type]:[identifier] [> child_selector]
```

- `:` separates the node type and its identifier.
- `>` expresses a **parent → child** AST relationship.
- Identifiers may include `*` wildcards.
- All matches are **case-sensitive**.
- Boolean operators (`&&`, `||`, `()`) are **not supported** in v0.1.

---

## ✅ Supported Node Types

| Type     | Description (Go AST)    |
| -------- | ----------------------- |
| `func`   | Function declaration    |
| `const`  | Constant declaration    |
| `var`    | Variable declaration    |
| `struct` | Struct type declaration |
| `field`  | Struct field            |
| `call`   | Function/method call    |
| `assign` | Assignment expression   |
| `if`     | If statement            |
| `import` | Import spec             |
| `block`  | Block of statements     |

---

## 🔡 Identifier Matching

| Pattern   | Meaning                               |
| --------- | ------------------------------------- |
| `Foo`     | Exact match                           |
| `Foo*`    | Starts with `Foo`                     |
| `*Foo`    | Ends with `Foo`                       |
| `*Foo*`   | Contains `Foo` anywhere               |
| `Foo*Bar` | Starts with `Foo` and ends with `Bar` |

Wildcard matching applies only to node names or declared types.

---

## ⛔ Negation

- Use `!` to negate a match:

  ```txt
  !func:Test*
  ```

- Only one negation per level is supported.
- Cannot negate nested conditions or chain `!` operators.

---

## 🌲 Parent/Child Relationships

Use `>` to express AST hierarchy:

```txt
func:Init > var:core.ModelConfig
```

- This matches any `func` named `Init` that **contains** a `var` declared as `core.ModelConfig`.

Chained traversal is supported:

```txt
func:Start > block > call:os.Getenv
```

---

## 🧠 Examples

| Query                              | Meaning                                             |
| ---------------------------------- | --------------------------------------------------- |
| `func:Init`                        | Exact function named `Init`                         |
| `func:Handle*`                     | Functions starting with `Handle`                    |
| `!func:Test*`                      | All functions except those starting with `Test`     |
| `func:* > var:core.ModelConfig`    | Any function containing a variable of this type     |
| `struct:* > field:Secret string`   | Structs with a field `Secret` of type `string`      |
| `func:Do > block > call:os.Getenv` | A function `Do` that contains a call to `os.Getenv` |

---

## 🧱 Internal Parsing (struct hint)

```go
type Query struct {
  Not        bool
  NodeType   string
  Identifier string
  Children   []Query
}
```

No complex operators. Each query is evaluated as a **tree of filters** with clear parent→child relationships.

---

## 🧼 Philosophy

MORFX DSL is designed to be:

- Minimal
- Case-sensitive
- Structurally precise
- Readable and composable (like CSS selectors, but for ASTs)

No ambiguity, no tolerance for bad casing, no overloaded syntax.

---

## 📌 Future Considerations

- Boolean operators (`&&`, `||`) — postponed
- Grouping (`(…)`) — postponed
- Regex matching — unlikely
- Configurable query aliases or saved patterns — maybe later
