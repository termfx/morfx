package core

import (
	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
)

// buildMatcher centralises the decision logic for regex vs AST based on rule cfg.
func buildMatcher(cfg model.ModificationConfig) (matcher.Matcher, error) {
	// Flags and preprocessing for regex are still handled in Manipulator.Apply.
	if cfg.UseAST {
		lang := cfg.Lang
		if lang == "" {
			lang = "go" // sensible default for Go projects
		}
		return matcher.NewAST(cfg.Pattern, lang)
	}
	// Regex path – cfg.Pattern may already include (?m) / (?s) etc.
	return matcher.NewRegex(cfg.Pattern)
}
