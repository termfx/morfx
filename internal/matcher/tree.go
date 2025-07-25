package matcher

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

// ASTMatcher uses Tree-sitter queries to match AST nodes.
type ASTMatcher struct {
	lang  *sitter.Language
	query *sitter.Query
}

// NewAST builds an ASTMatcher for the given query and language identifier.
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

// Find parses the source and returns the byte spans of the '@target' captures.
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
		// Filter out predicates before iterating captures
		match = cursor.FilterPredicates(match, src)
		for _, cap := range match.Captures {
			captureName := a.query.CaptureNameForId(cap.Index)
			// We only care about the final node designated as '@target'.
			if captureName == "target" {
				node := cap.Node
				res = append(res, Result{
					Start: int(node.StartByte()),
					End:   int(node.EndByte()),
				})
				// Found the target for this match, break inner loop to avoid duplicates.
				break
			}
		}
	}
	return res, nil
}

// UnsupportedLangError is returned when ResolveLanguage cannot map the name.
type UnsupportedLangError struct{ Lang string }

func (e *UnsupportedLangError) Error() string {
	return "unsupported language: " + e.Lang
}
