package matcher

import (
	sitter "github.com/smacker/go-tree-sitter"
	langGo "github.com/smacker/go-tree-sitter/golang"
)

// ResolveLanguage converts a short string ("go", "python", etc.) into the
// corresponding *sitter.Language. Add cases as you support more grammars.
func ResolveLanguage(name string) (*sitter.Language, bool) {
	switch name {
	case "go", "golang":
		return langGo.GetLanguage(), true
	default:
		return nil, false
	}
}
