package base

import (
	"context"
	"testing"

	"github.com/smacker/go-tree-sitter/golang"
)

func TestParserAdapterParse(t *testing.T) {
	adapter := newParserAdapter(golang.GetLanguage())
	if adapter == nil {
		t.Fatal("expected parser adapter")
	}

	source := []byte("package main\nfunc test() {}\n")

	tree, err := adapter.ParseCtx(context.Background(), nil, source)
	if err != nil {
		t.Fatalf("ParseCtx failed: %v", err)
	}
	if tree == nil {
		t.Fatal("ParseCtx returned nil tree")
	}
	defer tree.Close()

	tree2 := adapter.Parse(source)
	if tree2 == nil {
		t.Fatal("Parse returned nil tree")
	}
	defer tree2.Close()
}
