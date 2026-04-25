package base

import sitter "github.com/smacker/go-tree-sitter"

// Match is provider-local match metadata derived from a tree-sitter node.
type Match struct {
	Name      string
	Type      string
	NodeType  string
	StartByte uint32
	EndByte   uint32
	Line      uint32
	Column    uint32
	Captures  map[string]string
}

// Target is a provider-local transformation target that retains the parser node.
type Target struct {
	Match
	Node *sitter.Node
}

// NewTarget captures provider-local match metadata while keeping the parser node private to providers.
func NewTarget(node *sitter.Node, queryType, name string) Target {
	target := Target{
		Match: Match{
			Name: name,
			Type: queryType,
		},
		Node: node,
	}

	if node == nil {
		return target
	}

	target.NodeType = node.Type()
	target.StartByte = node.StartByte()
	target.EndByte = node.EndByte()
	target.Line = node.StartPoint().Row
	target.Column = node.StartPoint().Column

	return target
}
