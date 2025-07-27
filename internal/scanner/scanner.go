package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/garaekz/fileman/internal/provider"
)

// Scanner handles recursive directory traversal with filtering capabilities.
type Scanner struct {
	maxBytes       int64
	followSymlinks bool
	includeGlobs   []string
	excludeGlobs   []string
	noGitignore    bool
	provider       provider.LanguageProvider
	gitignore      *ignore.GitIgnore
}

// Config holds scanner configuration options.
type Config struct {
	MaxBytes       int64
	FollowSymlinks bool
	IncludeGlobs   []string
	ExcludeGlobs   []string
	NoGitignore    bool
	Provider       provider.LanguageProvider
}

// New creates a new scanner with the given configuration.
func New(cfg Config) *Scanner {
	s := &Scanner{
		maxBytes:       cfg.MaxBytes,
		followSymlinks: cfg.FollowSymlinks,
		includeGlobs:   cfg.IncludeGlobs,
		excludeGlobs:   cfg.ExcludeGlobs,
		noGitignore:    cfg.NoGitignore,
		provider:       cfg.Provider,
	}

	// Load .gitignore if not disabled
	if !cfg.NoGitignore {
		s.loadGitignore()
	}

	return s
}

// loadGitignore loads .gitignore patterns from the current directory and parent directories.
func (s *Scanner) loadGitignore() {
	// Start from current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return // Silently fail if we can't get current directory
	}

	// Look for .gitignore files up the directory tree
	var gitignoreFiles []string
	dir := cwd
	for {
		gitignorePath := filepath.Join(dir, ".gitignore")
		if _, err := os.Stat(gitignorePath); err == nil {
			gitignoreFiles = append(gitignoreFiles, gitignorePath)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root directory
		}
		dir = parent
	}

	// Load gitignore patterns (reverse order so closer .gitignore files take precedence)
	if len(gitignoreFiles) > 0 {
		// Reverse the slice to process from root to current directory
		for i := len(gitignoreFiles)/2 - 1; i >= 0; i-- {
			opp := len(gitignoreFiles) - 1 - i
			gitignoreFiles[i], gitignoreFiles[opp] = gitignoreFiles[opp], gitignoreFiles[i]
		}

		// CompileIgnoreFileAndLines expects first file as separate parameter
		if len(gitignoreFiles) == 1 {
			gitignore, err := ignore.CompileIgnoreFile(gitignoreFiles[0])
			if err == nil {
				s.gitignore = gitignore
			}
		} else {
			gitignore, err := ignore.CompileIgnoreFileAndLines(gitignoreFiles[0], gitignoreFiles[1:]...)
			if err == nil {
				s.gitignore = gitignore
			}
		}
	}
}

// ScanTargets processes a list of file and directory targets, returning a list of files to process.
func (s *Scanner) ScanTargets(ctx context.Context, targets []string) ([]string, error) {
	if len(targets) == 0 {
		// Default to current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
		targets = []string{cwd}
	}

	var allFiles []string
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		files, err := s.scanTarget(ctx, target)
		if err != nil {
			return nil, fmt.Errorf("scanning target %s: %w", target, err)
		}
		allFiles = append(allFiles, files...)
	}

	return s.deduplicateFiles(allFiles), nil
}

// scanTarget processes a single target (file or directory).
func (s *Scanner) scanTarget(ctx context.Context, target string) ([]string, error) {
	info, err := os.Lstat(target)
	if err != nil {
		return nil, fmt.Errorf("accessing target %s: %w", target, err)
	}

	// Handle symbolic links
	if info.Mode()&os.ModeSymlink != 0 {
		if !s.followSymlinks {
			return nil, nil // Skip symlinks unless explicitly following them
		}
		// Resolve the symlink
		resolved, err := filepath.EvalSymlinks(target)
		if err != nil {
			return nil, fmt.Errorf("resolving symlink %s: %w", target, err)
		}
		return s.scanTarget(ctx, resolved)
	}

	// Handle regular files
	if info.Mode().IsRegular() {
		if s.shouldProcessFile(target, info) {
			return []string{target}, nil
		}
		return nil, nil
	}

	// Handle directories
	if info.IsDir() {
		return s.scanDirectory(ctx, target)
	}

	return nil, nil // Skip other file types
}

// scanDirectory recursively scans a directory for files.
func (s *Scanner) scanDirectory(ctx context.Context, dir string) ([]string, error) {
	var files []string

	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Convert relative path to absolute
		fullPath := filepath.Join(dir, path)

		// Skip directories that should be ignored
		if d.IsDir() {
			if s.shouldSkipDirectory(path) {
				return fs.SkipDir
			}
			return nil
		}

		// Process regular files
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return fmt.Errorf("getting file info for %s: %w", fullPath, err)
			}

			if s.shouldProcessFile(fullPath, info) {
				files = append(files, fullPath)
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", dir, err)
	}

	return files, nil
}

// shouldProcessFile determines if a file should be processed based on various criteria.
func (s *Scanner) shouldProcessFile(path string, info os.FileInfo) bool {
	// Check gitignore patterns first (if enabled)
	if s.gitignore != nil {
		// Convert to relative path for gitignore matching
		if relPath, err := filepath.Rel(".", path); err == nil {
			if s.gitignore.MatchesPath(relPath) {
				return false
			}
		}
	}

	// Check file size limit
	if s.maxBytes > 0 && info.Size() > s.maxBytes {
		return false
	}

	// Check if file matches language provider's supported extensions
	if s.provider != nil {
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		found := slices.Contains(s.provider.Aliases(), ext)
		if !found {
			return false
		}
	}

	// Apply include/exclude glob patterns
	basename := filepath.Base(path)

	// If include patterns are specified, file must match at least one
	if len(s.includeGlobs) > 0 {
		matched := false
		for _, pattern := range s.includeGlobs {
			if match, _ := filepath.Match(pattern, basename); match {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// If exclude patterns are specified, file must not match any
	for _, pattern := range s.excludeGlobs {
		if match, _ := filepath.Match(pattern, basename); match {
			return false
		}
	}

	return true
}

// shouldSkipDirectory determines if a directory should be skipped during traversal.
func (s *Scanner) shouldSkipDirectory(path string) bool {
	// Check gitignore patterns first (if enabled)
	if s.gitignore != nil {
		// Convert to relative path for gitignore matching
		if relPath, err := filepath.Rel(".", path); err == nil {
			if s.gitignore.MatchesPath(relPath) {
				return true
			}
		}
	}

	dirname := filepath.Base(path)

	// Skip common non-source directories
	skipDirs := []string{".git", "vendor", "node_modules", "dist", "build", ".morfx"}
	if slices.Contains(skipDirs, dirname) {
		return true
	}

	// Skip hidden directories (except current directory)
	if strings.HasPrefix(dirname, ".") && dirname != "." {
		return true
	}

	return false
}

// deduplicateFiles removes duplicate file paths from the list.
func (s *Scanner) deduplicateFiles(files []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, file := range files {
		if !seen[file] {
			seen[file] = true
			result = append(result, file)
		}
	}

	return result
}
