package base

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

// parserAdapter owns tree-sitter parser construction and parse execution.
type parserAdapter struct {
	parser *sitter.Parser
}

func newParserAdapter(language *sitter.Language) *parserAdapter {
	parser := sitter.NewParser()
	parser.SetLanguage(language)
	return &parserAdapter{parser: parser}
}

func (p *parserAdapter) Parse(source []byte) *sitter.Tree {
	if p == nil || p.parser == nil {
		return nil
	}

	tree, err := p.parser.ParseCtx(context.TODO(), nil, source)
	if err != nil {
		return nil
	}

	return tree
}

func (p *parserAdapter) ParseCtx(ctx context.Context, oldTree *sitter.Tree, source []byte) (*sitter.Tree, error) {
	if p == nil || p.parser == nil {
		return nil, nil
	}

	return p.parser.ParseCtx(ctx, oldTree, source)
}
