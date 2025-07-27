# MORFX Staging Workflow

MORFX implements a two-step staging workflow similar to Git, providing safety and review capabilities for code transformations.

## Overview

By default, MORFX operates in **staging mode** - changes are prepared and stored in `.morfx/` directory for review before being applied to your files.

## Workflow Steps

### 1. Stage Changes (Default Behavior)

```bash
# Stage changes for review
morfx --query "func:main" --op insert-before --repl "// Added comment" src/
```

This will:

- Find matches for your query
- Prepare the changes
- Store them in `.morfx/` directory
- Show a diff preview of what would be applied

### 2. Review Staged Changes

The staging command shows you exactly what will be changed:

```
Staged 1 change(s) in .morfx/:

--- a/src/main.go
+++ b/src/main.go
 package main

+// Added comment
 func main() {
     println("Hello, World!")
 }

Run 'morfx --commit' to apply these changes.
```

### 3. Apply Staged Changes

```bash
# Apply all staged changes
morfx --commit
```

This will:

- Apply all changes stored in `.morfx/`
- Update your files
- Clean up the staging directory
- Show a summary of applied changes

## Alternative Modes

### Dry-Run Mode (Preview Only)

```bash
# Preview changes without staging
morfx --dry-run --query "func:main" --op delete src/
```

### Direct Mode (Skip Staging)

For simple operations, you can use `--commit` with a query to skip staging:

```bash
# Apply changes directly (not recommended for large changes)
morfx --commit --query "func:main" --op replace --repl "func main()" src/
```

## Safety Features

### File Integrity Checks

- SHA256 verification ensures files haven't changed between staging and commit
- Race condition detection prevents conflicts
- Atomic file operations ensure consistency

### Conflict Resolution

If a file has been modified since staging:

```
Error: file src/main.go has been modified since staging (hash mismatch)
```

In this case, you need to:

1. Clear staged changes: `rm -rf .morfx/`
2. Re-run your transformation
3. Review and commit the new changes

## Standard Input Support

You can pipe content for insertions:

```bash
# Read replacement content from stdin
echo "// This comment comes from stdin" | morfx --query "func:main" --op insert-before --stdin src/
```

## Best Practices

1. **Always review staged changes** before committing
2. **Use staging for large transformations** across multiple files
3. **Use dry-run for exploration** and testing queries
4. **Commit frequently** to avoid conflicts
5. **Clear staging** if you change your mind: `rm -rf .morfx/`

## Examples

### Basic Workflow

```bash
# 1. Stage changes
morfx --query "func:*" --op insert-after --repl "defer log.Println(\"function exit\")" ./

# 2. Review the diff output

# 3. Apply changes
morfx --commit
```

### With Standard Input

```bash
# 1. Stage changes with stdin content
cat header.txt | morfx --query "package:*" --op insert-before --stdin ./

# 2. Review and apply
morfx --commit
```

### Recursive with Filtering

```bash
# 1. Stage changes with gitignore filtering
morfx --query "struct:User" --op replace --repl "struct Person" --include "*.go" ./

# 2. Review and apply
morfx --commit
```
