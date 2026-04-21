package base

import (
	"context"

	sitter "github.com/smacker/go-tree-sitter"
)

// parserSeam defines the minimal parser operations used by base provider code.
type parserSeam interface {
	Parse([]byte) *sitter.Tree
	ParseCtx(context.Context, *sitter.Tree, []byte) (*sitter.Tree, error)
}

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

	tree, _ := p.parser.ParseCtx(context.TODO(), nil, source)
	return tree
}

func (p *parserAdapter) ParseCtx(ctx context.Context, oldTree *sitter.Tree, source []byte) (*sitter.Tree, error) {
	if p == nil || p.parser == nil {
		return nil, nil
	}

	return p.parser.ParseCtx(ctx, oldTree, source)
}
