package matcher

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

// ASTMatcher uses Tree‑sitter queries to match AST nodes.
// Pattern must be written in Tree‑sitter Query syntax.
type ASTMatcher struct {
	lang  *sitter.Language
	query *sitter.Query
}

// NewAST builds an ASTMatcher for the given query and language identifier.
// Example: NewAST("(function_declaration name: (identifier) @func)", "go")
func NewAST(pattern, langName string) (*ASTMatcher, error) {
	lang, ok := ResolveLanguage(langName)
	if !ok {
		return nil, &UnsupportedLangError{Lang: langName}
	}

	q, err := sitter.NewQuery([]byte(pattern), lang)
	if err != nil {
		return nil, err
	}
	return &ASTMatcher{lang: lang, query: q}, nil
}

// Find parses the source once and returns all capture spans (byte offsets).
// Each capture in the query is returned as a separate Result.
func (a *ASTMatcher) Find(src []byte) ([]Result, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(a.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(a.query, tree.RootNode())

	var res []Result
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, cap := range match.Captures {
			node := cap.Node
			res = append(res, Result{
				Start: int(node.StartByte()),
				End:   int(node.EndByte()),
			})
		}
	}
	return res, nil
}

// UnsupportedLangError is returned when ResolveLanguage cannot map the name.
type UnsupportedLangError struct{ Lang string }

func (e *UnsupportedLangError) Error() string {
	return "unsupported language: " + e.Lang
}
