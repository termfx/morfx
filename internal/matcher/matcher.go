package matcher

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/garaekz/fileman/internal/model"
)

// Matcher uses Tree-sitter queries to match AST nodes.
type Matcher struct {
	lang  *sitter.Language
	query *sitter.Query
}

// New builds an Matcher for the given query and language identifier.
func New(cfg *model.Config) (*Matcher, error) {
	language := cfg.Provider.GetSitterLanguage()

	q, err := sitter.NewQuery([]byte(cfg.Pattern), language)
	if err != nil {
		return nil, err
	}
	return &Matcher{lang: language, query: q}, nil
}

// Find parses the source and returns the Tree-sitter nodes of the '@target' captures.
func (a *Matcher) Find(src []byte) ([]*sitter.Node, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(a.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return nil, err
	}

	cursor := sitter.NewQueryCursor()
	cursor.Exec(a.query, tree.RootNode())

	var nodes []*sitter.Node
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
				nodes = append(nodes, cap.Node)
				// Found the target for this match, break inner loop to avoid duplicates.
				break
			}
		}
	}
	return nodes, nil
}
