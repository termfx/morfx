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

// Result represents a byte span (inclusive start, exclusive end) in the source.
type Result struct {
	Start int
	End   int
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

// Find parses the source and returns the byte spans of the '@target' captures.
func (a *Matcher) Find(src []byte) ([]Result, error) {
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
