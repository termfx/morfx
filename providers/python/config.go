package python

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"

	"github.com/termfx/morfx/core"
)

// Config implements LanguageConfig for Python
type Config struct{}

// Language identifier
func (c *Config) Language() string {
	return "python"
}

// Extensions supported
func (c *Config) Extensions() []string {
	return []string{".py", ".pyw", ".pyi"}
}

// GetLanguage returns tree-sitter language for Python
func (c *Config) GetLanguage() *sitter.Language {
	return python.GetLanguage()
}

// MapQueryTypeToNodeTypes maps query types to Python AST node types
func (c *Config) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function", "func":
		return []string{"function_definition", "async_function_definition"}
	case "class":
		return []string{"class_definition"}
	case "variable", "var":
		return []string{"assignment", "global_statement", "nonlocal_statement"}
	case "import":
		return []string{"import_statement", "import_from_statement"}
	case "decorator":
		return []string{"decorator"}
	default:
		// Try to use the query type directly as node type
		return []string{queryType}
	}
}

// ExtractNodeName extracts name from Python AST nodes
func (c *Config) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "function_definition", "async_function_definition", "class_definition":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil {
			return source[nameNode.StartByte():nameNode.EndByte()]
		}
	case "assignment":
		// Find left side of assignment
		if leftNode := node.ChildByFieldName("left"); leftNode != nil {
			if leftNode.Type() == "identifier" {
				return source[leftNode.StartByte():leftNode.EndByte()]
			}
		}
	case "import_statement":
		// Get first imported module
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "dotted_name" || child.Type() == "identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	case "import_from_statement":
		// Get module being imported from
		if moduleNode := node.ChildByFieldName("module_name"); moduleNode != nil {
			return source[moduleNode.StartByte():moduleNode.EndByte()]
		}
	case "decorator":
		// Get decorator name
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "identifier" || child.Type() == "attribute" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	}

	// Fallback: try to find first identifier child
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			return source[child.StartByte():child.EndByte()]
		}
	}

	return ""
}

// ValidateAssignment ensures assignments are actual variable definitions, not attribute assignments
func (c *Config) ValidateAssignment(node *sitter.Node, source, queryType string) bool {
	if node.Type() != "assignment" || queryType != "variable" {
		return true // Not assignment or not variable query
	}

	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return false
	}

	// Only accept simple identifiers and tuple/list unpacking for variable queries
	switch leftNode.Type() {
	case "identifier":
		return true // Simple variable assignment: x = 1
	case "tuple", "list", "pattern_list":
		return true // Tuple unpacking: a, b = 1, 2
	case "attribute":
		return false // Attribute assignment: self.x = 1
	case "subscript":
		return false // Array assignment: arr[0] = 1
	default:
		return false
	}
}

// IsExported checks if identifier is exported (in Python, typically non-underscore prefixed)
func (c *Config) IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	// In Python, single underscore prefix indicates "internal use"
	// Double underscore indicates "private" (name mangling)
	return !strings.HasPrefix(name, "_")
}

// ExpandMatches handles tuple unpacking and multiple assignments in Python
func (c *Config) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	switch node.Type() {
	case "assignment":
		return c.expandAssignment(node, source, query)
	default:
		name := c.ExtractNodeName(node, source)
		return []core.CodeMatch{{
			Node:      node,
			Name:      name,
			Type:      query.Type,
			NodeType:  node.Type(),
			StartByte: node.StartByte(),
			EndByte:   node.EndByte(),
			Line:      node.StartPoint().Row,
			Column:    node.StartPoint().Column,
		}}
	}
}

func (c *Config) expandAssignment(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	leftNode := node.ChildByFieldName("left")
	if leftNode == nil {
		return matches
	}

	switch leftNode.Type() {
	case "identifier":
		name := source[leftNode.StartByte():leftNode.EndByte()]
		matches = append(matches, core.CodeMatch{
			Node:      leftNode,
			Name:      name,
			Type:      query.Type,
			NodeType:  "identifier",
			StartByte: leftNode.StartByte(),
			EndByte:   leftNode.EndByte(),
			Line:      leftNode.StartPoint().Row,
			Column:    leftNode.StartPoint().Column,
		})
	case "tuple", "list":
		matches = append(matches, c.expandTupleOrList(leftNode, source, query)...)
	case "pattern_list":
		matches = append(matches, c.expandPatternList(leftNode, source, query)...)
	}

	return matches
}

func (c *Config) expandTupleOrList(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, core.CodeMatch{
				Node:      child,
				Name:      name,
				Type:      query.Type,
				NodeType:  "identifier",
				StartByte: child.StartByte(),
				EndByte:   child.EndByte(),
				Line:      child.StartPoint().Row,
				Column:    child.StartPoint().Column,
			})
		}
	}

	return matches
}

func (c *Config) expandPatternList(node *sitter.Node, source string, query core.AgentQuery) []core.CodeMatch {
	var matches []core.CodeMatch

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" {
			name := source[child.StartByte():child.EndByte()]
			matches = append(matches, core.CodeMatch{
				Node:      child,
				Name:      name,
				Type:      query.Type,
				NodeType:  "identifier",
				StartByte: child.StartByte(),
				EndByte:   child.EndByte(),
				Line:      child.StartPoint().Row,
				Column:    child.StartPoint().Column,
			})
		}
	}

	return matches
}
