# Morfx Demo

Interactive demonstration of Morfx AST transformations with real code modifications.

## Features

- **Real transformations**: Actual file modifications using morfx tools
- **Auto backup/restore**: Safe demo environment with automatic cleanup
- **Multi-language support**: Go, PHP, JavaScript, TypeScript
- **Visual diffs**: Clear before/after comparisons
- **Interactive mode**: Step through transformations or run all

## Structure

```
demo/
├── README.md          # This file
├── cmd/
│   └── main.go        # CLI implementation
└── fixtures/          # Sample files for transformation
    ├── example.go     # Go code samples
    ├── example.php    # PHP code samples  
    ├── example.js     # JavaScript code samples
    └── example.ts     # TypeScript code samples
```

## Usage

```bash
# Run interactive demo
go run ./demo/cmd run

# Run specific scenario
go run ./demo/cmd run --scenario go-query

# List available scenarios
go run ./demo/cmd list
```

## Demo Scenarios

- **go-query**: Query functions with 'User' pattern
- **php-replace**: Replace updateEmail method
- **js-insert**: Insert validation function
- **ts-delete**: Delete deprecated function

## How it Works

1. **Backup**: Creates temporary backup of all sample files
2. **Transform**: Applies real morfx transformations to actual files
3. **Display**: Shows before/after diffs with syntax highlighting
4. **Restore**: Automatically restores original files
5. **Repeat**: Ready for next demonstration

The demo uses the morfx tools available in this project to perform actual AST transformations on real code files, providing a realistic demonstration of the system's capabilities.
