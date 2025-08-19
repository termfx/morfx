package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/termfx/morfx/internal/model"
	"github.com/termfx/morfx/internal/provider"
	"github.com/termfx/morfx/internal/registry"
	"github.com/termfx/morfx/internal/scanner"
)

// checkCommit checks if the commit flag is set and returns a configuration for commit operation.
func checkCommit(fs *pflag.FlagSet) (*model.Config, bool) {
	if fs.Changed("commit") {
		return &model.Config{
			RuleID:      "cli-commit",
			Operation:   model.OpCommit, // Commit operation
			DryRun:      false,          // Commit always writes changes
			Interactive: false,          // No interactive mode for commit
		}, true
	}
	return nil, false
}

// checkQuery checks if the query flag is set and returns a configuration for query operation.
func checkQuery(fs *pflag.FlagSet) (string, bool) {
	if fs.Changed("query") {
		query, _ := fs.GetString("query")
		return query, true
	}
	return "", false
}

// resolveTargets resolves the command-line arguments into a list of file or directory targets.
func resolveTargets(fs *pflag.FlagSet) ([]string, error) {
	targets := fs.Args()
	if len(targets) > 0 {
		return targets, nil
	}

	root, err := fs.GetString("root")
	if err != nil {
		return nil, err
	}

	if root != "" {
		return []string{root}, nil
	}

	// Default to current working directory if no targets or root specified
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return []string{cwd}, nil
}

// resolveProviderAndFiles resolves the language provider and files based on command-line flags.
func resolveProviderAndFiles(fs *pflag.FlagSet, targets []string) (provider.LanguageProvider, []string, error) {
	codeLang, err := fs.GetString("lang")
	if err != nil {
		return nil, nil, fmt.Errorf("getting language: %w", err)
	}
	cfg := buildScannerConfig(fs)

	var prov provider.LanguageProvider
	var files []string
	if codeLang != "" {
		// Try to get provider from registry
		// (assumes registry is initialized elsewhere)
		prov, err = registry.GetProvider(codeLang)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving language provider: %w", err)
		}
	}
	// At this point provider is either resolved or nil
	cfg.Provider = prov
	s := scanner.New(cfg)

	files, err = s.ScanTargets(context.Background(), targets)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning targets: %w", err)
	}

	// Either we've found the provider or not, we still don't have files
	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no files found to process")
	}

	// If not resolved by now, resolve provider based on files
	if prov == nil {
		// Try to detect from file extension
		// (assumes registry is initialized elsewhere)
		if len(files) > 0 {
			ext := filepath.Ext(files[0])
			prov, err = registry.GetProviderByExtension(ext)
		} else {
			err = errors.New("no files provided")
		}
		if err != nil {
			return nil, nil, err
		}
	}

	return prov, files, nil
}

// resolveReplacement resolves the replacement text from command-line flags.
func resolveReplacement(fs *pflag.FlagSet) (string, error) {
	rep, err := fs.GetString("repl")
	if err != nil {
		return "", fmt.Errorf("getting replacement text: %w", err)
	}

	if rep != "" {
		return rep, nil
	}

	// If stdin mode is enabled, read from stdin
	if ok, _ := fs.GetBool("stdin"); ok {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			bytes, err := io.ReadAll(os.Stdin)
			if err != nil || len(bytes) == 0 {
				return "", fmt.Errorf("failed to read from stdin: %w", err)
			}
			return string(bytes), nil
		} else {
			return "", errors.New("stdin mode requires input from stdin")
		}
	}

	return "", errors.New("replacement text is required, use --repl or --stdin")
}

func buildScannerConfig(fs *pflag.FlagSet) scanner.Config {
	maxBytes, _ := fs.GetInt64("max-bytes")
	includeGlobs, _ := fs.GetStringSlice("include")
	excludeGlobs, _ := fs.GetStringSlice("exclude")
	return scanner.Config{
		MaxBytes:       maxBytes | 5*1024*1024, // Default to 5MB if not set
		FollowSymlinks: fs.Changed("follow-symlinks"),
		IncludeGlobs:   includeGlobs,
		ExcludeGlobs:   excludeGlobs,
		NoGitignore:    fs.Changed("no-gitignore"),
	}
}
