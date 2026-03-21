package core

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/termfx/morfx/providers/catalog"
)

// FileWalker provides high-performance parallel file system traversal
type FileWalker struct {
	workers    int
	bufferSize int
}

// NewFileWalker creates a new file walker optimized for performance
func NewFileWalker() *FileWalker {
	return &FileWalker{
		workers:    runtime.NumCPU() * 2, // 2x CPU cores for I/O bound work
		bufferSize: 1000,                 // Channel buffer size
	}
}

// WalkResult represents a discovered file
type WalkResult struct {
	Path     string
	Info     fs.FileInfo
	Language string
	Error    error
}

// Walk performs parallel directory traversal with pattern matching
func (fw *FileWalker) Walk(ctx context.Context, scope FileScope) (<-chan WalkResult, error) {
	// Validate scope
	if err := fw.validateScope(scope); err != nil {
		return nil, err
	}

	// Create channels
	results := make(chan WalkResult, fw.bufferSize)
	paths := make(chan string, fw.bufferSize)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < fw.workers; i++ {
		wg.Add(1)
		go fw.worker(ctx, paths, results, scope, &wg)
	}

	// Start directory scanner in separate goroutine
	go func() {
		defer close(paths)
		processed := 0
		var visited map[string]struct{}
		if scope.FollowSymlinks {
			visited = make(map[string]struct{})
			if resolved, err := filepath.EvalSymlinks(scope.Path); err == nil {
				visited[resolved] = struct{}{}
			} else {
				visited[scope.Path] = struct{}{}
			}
		}
		fw.scanDirectory(ctx, scope.Path, scope, paths, 0, &processed, visited)
	}()

	// Close results when all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	return results, nil
}

// worker processes file paths in parallel
func (fw *FileWalker) worker(
	ctx context.Context,
	paths <-chan string,
	results chan<- WalkResult,
	scope FileScope,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case path, ok := <-paths:
			if !ok {
				return
			}

			// Process file
			result := fw.processFile(path, scope)

			select {
			case <-ctx.Done():
				return
			case results <- result:
			}
		}
	}
}

// scanDirectory recursively discovers files matching patterns
func (fw *FileWalker) scanDirectory(
	ctx context.Context,
	dirPath string,
	scope FileScope,
	paths chan<- string,
	depth int,
	processed *int,
	visited map[string]struct{},
) {
	if scope.MaxFiles > 0 && *processed >= scope.MaxFiles {
		return
	}
	// Check context cancellation
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Check max depth
	if scope.MaxDepth > 0 && depth > scope.MaxDepth {
		return
	}

	// Read directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return // Skip directories we can't read
	}

	for _, entry := range entries {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		fullPath := filepath.Join(dirPath, entry.Name())

		// Skip excluded patterns
		if fw.isExcluded(fullPath, scope.Exclude) {
			continue
		}

		// Handle symlinked directories when allowed
		if entry.Type()&os.ModeSymlink != 0 && scope.FollowSymlinks {
			resolvedPath, err := filepath.EvalSymlinks(fullPath)
			if err != nil || resolvedPath == "" {
				continue
			}

			info, err := os.Stat(resolvedPath)
			if err != nil {
				continue
			}

			if info.IsDir() {
				if visited != nil {
					if _, seen := visited[resolvedPath]; seen {
						continue
					}
					visited[resolvedPath] = struct{}{}
				}
				fw.scanDirectory(ctx, fullPath, scope, paths, depth+1, processed, visited)
				continue
			}
		}

		if entry.IsDir() {
			if visited != nil {
				realPath := fullPath
				if resolved, err := filepath.EvalSymlinks(fullPath); err == nil && resolved != "" {
					realPath = resolved
				}
				if _, seen := visited[realPath]; seen {
					continue
				}
				visited[realPath] = struct{}{}
			}

			// Recurse into subdirectory
			fw.scanDirectory(ctx, fullPath, scope, paths, depth+1, processed, visited)
			continue
		}

		// Check if file matches include patterns
		if fw.isIncluded(fullPath, scope.Include) {
			if scope.MaxFiles > 0 && *processed >= scope.MaxFiles {
				return
			}
			select {
			case <-ctx.Done():
				return
			case paths <- fullPath:
				*processed++
			}
		}
	}
}

// processFile analyzes a single file and creates WalkResult
func (fw *FileWalker) processFile(path string, scope FileScope) WalkResult {
	info, err := os.Stat(path)
	if err != nil {
		return WalkResult{Path: path, Error: err}
	}

	// Detect language
	language := scope.Language
	if language == "" {
		language = fw.detectLanguage(path)
	}

	return WalkResult{
		Path:     path,
		Info:     info,
		Language: language,
	}
}

// detectLanguage determines programming language from file extension
func (fw *FileWalker) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	if info, ok := catalog.LookupByExtension(ext); ok {
		return info.ID
	}

	languageMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".mjs":   "javascript",
		".cjs":   "javascript",
		".java":  "java",
		".cpp":   "cpp",
		".c":     "c",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".phtml": "php",
		".php4":  "php",
		".php5":  "php",
		".phps":  "php",
		".rs":    "rust",
		".kt":    "kotlin",
		".swift": "swift",
		".dart":  "dart",
		".scala": "scala",
		".clj":   "clojure",
		".ml":    "ocaml",
		".hs":    "haskell",
		".elm":   "elm",
		".ex":    "elixir",
		".erl":   "erlang",
		".pyw":   "python",
		".pyi":   "python",
		".d.ts":  "typescript",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

// isIncluded checks if file matches include patterns
func (fw *FileWalker) isIncluded(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return true // Include all if no patterns specified
	}

	for _, pattern := range patterns {
		if fw.matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

// isExcluded checks if file matches exclude patterns
func (fw *FileWalker) isExcluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if fw.matchPattern(path, pattern) {
			return true
		}
	}
	return false
}

// matchPattern performs robust glob-style pattern matching with ** support
func (fw *FileWalker) matchPattern(path, pattern string) bool {
	// Direct match with doublestar
	if matched, err := doublestar.PathMatch(pattern, path); err == nil && matched {
		return true
	}

	// Try basename for simple patterns without path separators
	if !strings.Contains(pattern, "/") {
		basename := filepath.Base(path)
		if matched, err := doublestar.PathMatch(pattern, basename); err == nil && matched {
			return true
		}
	}

	return false
}

// validateScope validates FileScope parameters
func (fw *FileWalker) validateScope(scope FileScope) error {
	if scope.Path == "" {
		return fmt.Errorf("path is required")
	}

	// Check if path exists and is accessible
	info, err := os.Stat(scope.Path)
	if err != nil {
		return fmt.Errorf("cannot access path %s: %w", scope.Path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path %s is not a directory", scope.Path)
	}

	return nil
}

// FastScan performs ultra-fast file discovery without full stat() calls
func (fw *FileWalker) FastScan(ctx context.Context, scope FileScope) ([]string, error) {
	var files []string
	var mu sync.Mutex

	results, err := fw.Walk(ctx, scope)
	if err != nil {
		return nil, err
	}

	for result := range results {
		if result.Error != nil {
			continue // Skip files with errors in fast scan
		}

		mu.Lock()
		files = append(files, result.Path)
		mu.Unlock()
	}

	return files, nil
}

// GetLanguageStats returns statistics about discovered files by language
func (fw *FileWalker) GetLanguageStats(ctx context.Context, scope FileScope) (map[string]int, error) {
	stats := make(map[string]int)
	var mu sync.Mutex

	results, err := fw.Walk(ctx, scope)
	if err != nil {
		return nil, err
	}

	for result := range results {
		if result.Error != nil {
			continue
		}

		mu.Lock()
		stats[result.Language]++
		mu.Unlock()
	}

	return stats, nil
}
