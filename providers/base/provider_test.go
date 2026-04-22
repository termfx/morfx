package base

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	tsjavascript "github.com/smacker/go-tree-sitter/javascript"
	tsphp "github.com/smacker/go-tree-sitter/php"
	tspython "github.com/smacker/go-tree-sitter/python"
	tstypescript "github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/oxhq/morfx/core"
)

// mockConfig implements LanguageConfig for testing
type mockConfig struct {
	language   string
	extensions []string
}

type legacyMockConfig struct {
	language   string
	extensions []string
}

type multiBindingConfig struct{}

type typeSpecValidatorConfig struct{}
type ifaceAliasConfig struct{}
type pythonVarAliasConfig struct{}
type phpPropertyConfig struct{}
type jsConstructorConfig struct{}
type tsSemanticMethodConfig struct{}
type phpConstructorConfig struct{}
type jsOverlapVariableConfig struct{}
type phpOverlapVariableConfig struct{}

func (m *mockConfig) Language() string {
	return m.language
}

func (m *legacyMockConfig) Language() string {
	return m.language
}

func (m *mockConfig) Extensions() []string {
	return m.extensions
}

func (m *legacyMockConfig) Extensions() []string {
	return m.extensions
}

func (m *mockConfig) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

func (m *legacyMockConfig) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

func (m *mockConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function":
		return []string{"function_declaration"}
	case "struct":
		return []string{"type_spec"}
	case "variable":
		return []string{"var_declaration"}
	default:
		return []string{queryType}
	}
}

func (m *legacyMockConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	return (&mockConfig{}).MapQueryTypeToNodeTypes(queryType)
}

func (m *mockConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}
	return ""
}

func (m *legacyMockConfig) ExtractNodeName(node *sitter.Node, source string) string {
	return (&mockConfig{}).ExtractNodeName(node, source)
}

func (m *mockConfig) IsExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func (m *legacyMockConfig) IsExported(name string) bool {
	return (&mockConfig{}).IsExported(name)
}

func (m *mockConfig) SupportedQueryTypes() []string {
	return []string{"function", "struct", "variable"}
}

func (m *legacyMockConfig) SupportedQueryTypes() []string {
	return (&mockConfig{}).SupportedQueryTypes()
}

func (m *mockConfig) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []Target {
	name := m.ExtractNodeName(node, source)
	return []Target{NewTarget(node, query.Type, name)}
}

func (c *multiBindingConfig) Language() string {
	return "go"
}

func (c *multiBindingConfig) Extensions() []string {
	return []string{".go"}
}

func (c *multiBindingConfig) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

func (c *multiBindingConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	if queryType == "variable" {
		return []string{"var_declaration"}
	}
	return []string{queryType}
}

func (c *multiBindingConfig) ExtractNodeName(node *sitter.Node, source string) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "var_spec" {
			for j := 0; j < int(child.ChildCount()); j++ {
				identifier := child.Child(j)
				if identifier.Type() == "identifier" {
					return source[identifier.StartByte():identifier.EndByte()]
				}
			}
		}
	}
	return ""
}

func (c *multiBindingConfig) IsExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func (c *multiBindingConfig) SupportedQueryTypes() []string {
	return []string{"variable"}
}

func (c *multiBindingConfig) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []Target {
	var targets []Target
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "var_spec" {
			continue
		}
		for j := 0; j < int(child.ChildCount()); j++ {
			identifier := child.Child(j)
			if identifier.Type() == "identifier" {
				name := source[identifier.StartByte():identifier.EndByte()]
				targets = append(targets, NewTarget(identifier, query.Type, name))
			}
		}
	}
	return targets
}

func (c *typeSpecValidatorConfig) Language() string {
	return "go"
}

func (c *typeSpecValidatorConfig) Extensions() []string {
	return []string{".go"}
}

func (c *typeSpecValidatorConfig) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

func (c *typeSpecValidatorConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "struct", "interface":
		return []string{"type_spec"}
	default:
		return []string{queryType}
	}
}

func (c *typeSpecValidatorConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}
	return ""
}

func (c *typeSpecValidatorConfig) IsExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func (c *typeSpecValidatorConfig) SupportedQueryTypes() []string {
	return []string{"struct", "interface"}
}

func (c *typeSpecValidatorConfig) ValidateTypeSpec(node *sitter.Node, source, queryType string) bool {
	if node.Type() != "type_spec" {
		return true
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return false
	}

	switch queryType {
	case "struct":
		return typeNode.Type() == "struct_type"
	case "interface":
		return typeNode.Type() == "interface_type"
	default:
		return true
	}
}

func (c *ifaceAliasConfig) Language() string {
	return "go"
}

func (c *ifaceAliasConfig) Extensions() []string {
	return []string{".go"}
}

func (c *ifaceAliasConfig) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

func (c *ifaceAliasConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "interface", "iface":
		return []string{"type_spec"}
	default:
		return []string{queryType}
	}
}

func (c *ifaceAliasConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}
	return ""
}

func (c *ifaceAliasConfig) IsExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func (c *ifaceAliasConfig) SupportedQueryTypes() []string {
	return []string{"interface", "iface"}
}

func (c *ifaceAliasConfig) ValidateTypeSpec(node *sitter.Node, source, queryType string) bool {
	if node.Type() != "type_spec" {
		return true
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return false
	}

	switch queryType {
	case "interface", "iface":
		return typeNode.Type() == "interface_type"
	default:
		return true
	}
}

func (c *pythonVarAliasConfig) Language() string {
	return "python"
}

func (c *pythonVarAliasConfig) Extensions() []string {
	return []string{".py"}
}

func (c *pythonVarAliasConfig) GetLanguage() *sitter.Language {
	return tspython.GetLanguage()
}

func (c *pythonVarAliasConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "variable", "var":
		return []string{"assignment", "augmented_assignment"}
	default:
		return []string{queryType}
	}
}

func (c *pythonVarAliasConfig) ExtractNodeName(node *sitter.Node, source string) string {
	left := node.ChildByFieldName("left")
	if left == nil {
		return ""
	}

	switch left.Type() {
	case "identifier":
		return source[left.StartByte():left.EndByte()]
	case "attribute":
		for i := 0; i < int(left.ChildCount()); i++ {
			child := left.Child(i)
			if child.Type() == "identifier" {
				return source[child.StartByte():child.EndByte()]
			}
		}
	}

	return ""
}

func (c *pythonVarAliasConfig) IsExported(name string) bool {
	return !strings.HasPrefix(name, "_")
}

func (c *pythonVarAliasConfig) SupportedQueryTypes() []string {
	return []string{"variable", "var"}
}

func (c *pythonVarAliasConfig) ValidateAssignment(node *sitter.Node, source, queryType string) bool {
	if node.Type() != "assignment" && node.Type() != "augmented_assignment" {
		return true
	}

	if queryType != "variable" && queryType != "var" {
		return true
	}

	left := node.ChildByFieldName("left")
	if left == nil {
		return false
	}

	switch left.Type() {
	case "identifier", "tuple", "list", "pattern_list":
		return true
	case "attribute", "subscript":
		return false
	default:
		return false
	}
}

func (c *phpPropertyConfig) Language() string {
	return "php"
}

func (c *phpPropertyConfig) Extensions() []string {
	return []string{".php"}
}

func (c *phpPropertyConfig) GetLanguage() *sitter.Language {
	return tsphp.GetLanguage()
}

func (c *phpPropertyConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "property":
		return []string{"property_declaration"}
	default:
		return []string{queryType}
	}
}

func (c *phpPropertyConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if variableNode := c.firstVariableNode(node); variableNode != nil {
		return strings.TrimPrefix(source[variableNode.StartByte():variableNode.EndByte()], "$")
	}
	return ""
}

func (c *phpPropertyConfig) IsExported(name string) bool {
	return !strings.HasPrefix(name, "_")
}

func (c *phpPropertyConfig) SupportedQueryTypes() []string {
	return []string{"property"}
}

func (c *phpPropertyConfig) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []Target {
	var targets []Target
	for _, child := range c.variableNodes(node) {
		name := strings.TrimPrefix(source[child.StartByte():child.EndByte()], "$")
		targets = append(targets, NewTarget(child, query.Type, name))
	}
	return targets
}

func (c *phpPropertyConfig) firstVariableNode(node *sitter.Node) *sitter.Node {
	nodes := c.variableNodes(node)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

func (c *phpPropertyConfig) variableNodes(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}

	var nodes []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(current *sitter.Node) {
		if current == nil {
			return
		}
		if current.Type() == "variable_name" {
			nodes = append(nodes, current)
			return
		}
		for i := 0; i < int(current.ChildCount()); i++ {
			walk(current.Child(i))
		}
	}
	walk(node)
	return nodes
}

func (c *jsConstructorConfig) Language() string {
	return "javascript"
}

func (c *jsConstructorConfig) Extensions() []string {
	return []string{".js"}
}

func (c *jsConstructorConfig) GetLanguage() *sitter.Language {
	return tsjavascript.GetLanguage()
}

func (c *jsConstructorConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "constructor", "ctor":
		return []string{"method_definition"}
	default:
		return []string{queryType}
	}
}

func (c *jsConstructorConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if keyNode := childByFieldNames(node, "name", "key"); keyNode != nil {
		return source[keyNode.StartByte():keyNode.EndByte()]
	}
	return ""
}

func (c *jsConstructorConfig) IsExported(name string) bool {
	return true
}

func (c *jsConstructorConfig) SupportedQueryTypes() []string {
	return []string{"constructor", "ctor"}
}

func (c *jsConstructorConfig) ValidateQueryNode(node *sitter.Node, source, queryType string) bool {
	if queryType != "constructor" && queryType != "ctor" {
		return true
	}
	return node.Type() == "method_definition" && c.ExtractNodeName(node, source) == "constructor"
}

func (c *tsSemanticMethodConfig) Language() string {
	return "typescript"
}

func (c *tsSemanticMethodConfig) Extensions() []string {
	return []string{".ts"}
}

func (c *tsSemanticMethodConfig) GetLanguage() *sitter.Language {
	return tstypescript.GetLanguage()
}

func (c *tsSemanticMethodConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "getter", "setter", "accessor":
		return []string{"method_definition", "method_signature"}
	case "constructor", "ctor":
		return []string{"method_definition"}
	default:
		return []string{queryType}
	}
}

func (c *tsSemanticMethodConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if keyNode := node.ChildByFieldName("key"); keyNode != nil {
		return source[keyNode.StartByte():keyNode.EndByte()]
	}
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "property_identifier" {
			return source[child.StartByte():child.EndByte()]
		}
	}
	return ""
}

func (c *tsSemanticMethodConfig) IsExported(name string) bool {
	return true
}

func (c *tsSemanticMethodConfig) SupportedQueryTypes() []string {
	return []string{"getter", "setter", "accessor", "constructor", "ctor"}
}

func (c *tsSemanticMethodConfig) ValidateQueryNode(node *sitter.Node, source, queryType string) bool {
	switch queryType {
	case "constructor", "ctor":
		return node.Type() == "method_definition" && c.ExtractNodeName(node, source) == "constructor"
	case "getter":
		return testMemberKeywordBeforeName(node, source, "get")
	case "setter":
		return testMemberKeywordBeforeName(node, source, "set")
	case "accessor":
		return testMemberKeywordBeforeName(node, source, "accessor")
	default:
		return true
	}
}

func (c *phpConstructorConfig) Language() string {
	return "php"
}

func (c *phpConstructorConfig) Extensions() []string {
	return []string{".php"}
}

func (c *phpConstructorConfig) GetLanguage() *sitter.Language {
	return tsphp.GetLanguage()
}

func (c *phpConstructorConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "constructor", "ctor":
		return []string{"method_declaration"}
	default:
		return []string{queryType}
	}
}

func (c *phpConstructorConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}
	return ""
}

func (c *phpConstructorConfig) IsExported(name string) bool {
	return true
}

func (c *phpConstructorConfig) SupportedQueryTypes() []string {
	return []string{"constructor", "ctor"}
}

func (c *phpConstructorConfig) ValidateQueryNode(node *sitter.Node, source, queryType string) bool {
	if queryType != "constructor" && queryType != "ctor" {
		return true
	}
	return node.Type() == "method_declaration" && strings.EqualFold(c.ExtractNodeName(node, source), "__construct")
}

func (c *jsOverlapVariableConfig) Language() string {
	return "javascript"
}

func (c *jsOverlapVariableConfig) Extensions() []string {
	return []string{".js"}
}

func (c *jsOverlapVariableConfig) GetLanguage() *sitter.Language {
	return tsjavascript.GetLanguage()
}

func (c *jsOverlapVariableConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "variable":
		return []string{"lexical_declaration", "variable_declarator"}
	default:
		return []string{queryType}
	}
}

func (c *jsOverlapVariableConfig) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "lexical_declaration":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_declarator" {
				return c.ExtractNodeName(child, source)
			}
		}
	case "variable_declarator":
		if idNode := childByFieldNames(node, "name", "id"); idNode != nil {
			return source[idNode.StartByte():idNode.EndByte()]
		}
	}

	return ""
}

func (c *jsOverlapVariableConfig) IsExported(name string) bool {
	return true
}

func (c *jsOverlapVariableConfig) SupportedQueryTypes() []string {
	return []string{"variable"}
}

func (c *jsOverlapVariableConfig) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []Target {
	switch node.Type() {
	case "lexical_declaration":
		var targets []Target
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_declarator" {
				targets = append(targets, c.ExpandMatches(child, source, query)...)
			}
		}
		return targets
	case "variable_declarator":
		if idNode := childByFieldNames(node, "name", "id"); idNode != nil {
			name := source[idNode.StartByte():idNode.EndByte()]
			return []Target{NewTarget(idNode, query.Type, name)}
		}
	}

	return nil
}

func (c *phpOverlapVariableConfig) Language() string {
	return "php"
}

func (c *phpOverlapVariableConfig) Extensions() []string {
	return []string{".php"}
}

func (c *phpOverlapVariableConfig) GetLanguage() *sitter.Language {
	return tsphp.GetLanguage()
}

func (c *phpOverlapVariableConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "variable":
		return []string{"assignment_expression", "simple_parameter", "property_declaration", "variable_name"}
	default:
		return []string{queryType}
	}
}

func (c *phpOverlapVariableConfig) ExtractNodeName(node *sitter.Node, source string) string {
	switch node.Type() {
	case "assignment_expression":
		if left := c.variableTargetNode(node); left != nil {
			return strings.TrimPrefix(source[left.StartByte():left.EndByte()], "$")
		}
	case "simple_parameter":
		if nameNode := c.variableTargetNode(node); nameNode != nil {
			return strings.TrimPrefix(source[nameNode.StartByte():nameNode.EndByte()], "$")
		}
	case "property_declaration":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_name" {
				return strings.TrimPrefix(source[child.StartByte():child.EndByte()], "$")
			}
		}
	case "variable_name":
		return strings.TrimPrefix(source[node.StartByte():node.EndByte()], "$")
	}

	return ""
}

func (c *phpOverlapVariableConfig) IsExported(name string) bool {
	return true
}

func (c *phpOverlapVariableConfig) SupportedQueryTypes() []string {
	return []string{"variable"}
}

func (c *phpOverlapVariableConfig) ExpandMatches(node *sitter.Node, source string, query core.AgentQuery) []Target {
	switch node.Type() {
	case "assignment_expression":
		if left := c.variableTargetNode(node); left != nil {
			name := strings.TrimPrefix(source[left.StartByte():left.EndByte()], "$")
			return []Target{NewTarget(left, query.Type, name)}
		}
	case "simple_parameter":
		if nameNode := c.variableTargetNode(node); nameNode != nil {
			name := strings.TrimPrefix(source[nameNode.StartByte():nameNode.EndByte()], "$")
			return []Target{NewTarget(nameNode, query.Type, name)}
		}
	case "property_declaration":
		var targets []Target
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_name" {
				name := strings.TrimPrefix(source[child.StartByte():child.EndByte()], "$")
				targets = append(targets, NewTarget(child, query.Type, name))
			}
		}
		return targets
	case "variable_name":
		name := strings.TrimPrefix(source[node.StartByte():node.EndByte()], "$")
		return []Target{NewTarget(node, query.Type, name)}
	}

	return nil
}

func (c *phpOverlapVariableConfig) variableTargetNode(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}

	switch node.Type() {
	case "assignment_expression":
		if left := node.ChildByFieldName("left"); left != nil && left.Type() == "variable_name" {
			return left
		}
	case "simple_parameter":
		if nameNode := node.ChildByFieldName("name"); nameNode != nil && nameNode.Type() == "variable_name" {
			return nameNode
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "variable_name" {
				return child
			}
		}
	}

	return nil
}

func testMemberKeywordBeforeName(node *sitter.Node, source, keyword string) bool {
	if node == nil {
		return false
	}

	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = node.ChildByFieldName("key")
	}
	if nameNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child == nil {
				continue
			}
			switch child.Type() {
			case "property_identifier", "private_property_identifier", "identifier":
				nameNode = child
				i = int(node.ChildCount())
			}
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if nameNode != nil && child.StartByte() == nameNode.StartByte() && child.EndByte() == nameNode.EndByte() {
			return false
		}

		text := strings.TrimSpace(source[child.StartByte():child.EndByte()])
		if text == keyword {
			return true
		}
	}

	return false
}

func childByFieldNames(node *sitter.Node, names ...string) *sitter.Node {
	if node == nil {
		return nil
	}
	for _, name := range names {
		if child := node.ChildByFieldName(name); child != nil {
			return child
		}
	}
	return nil
}

func newTestProvider() *Provider {
	config := &mockConfig{
		language:   "go",
		extensions: []string{".go"},
	}
	return New(config)
}

// TestNew tests provider creation
func TestNew(t *testing.T) {
	config := &mockConfig{
		language:   "go",
		extensions: []string{".go"},
	}

	provider := New(config)
	if provider == nil {
		t.Fatal("New returned nil")
	}

	if provider.config != config {
		t.Error("Config not set properly")
	}

	if provider.pool == nil {
		t.Error("Parser pool not initialized")
	}

	if provider.cache == nil {
		t.Error("Cache not initialized")
	}
}

// TestLanguage tests language getter
func TestLanguage(t *testing.T) {
	provider := newTestProvider()
	if provider.Language() != "go" {
		t.Errorf("Expected 'go', got '%s'", provider.Language())
	}
}

// TestExtensions tests extensions getter
func TestExtensions(t *testing.T) {
	provider := newTestProvider()
	extensions := provider.Extensions()

	if len(extensions) != 1 || extensions[0] != ".go" {
		t.Errorf("Expected ['.go'], got %v", extensions)
	}
}

// TestQuery tests code element queries
func TestQuery(t *testing.T) {
	provider := newTestProvider()

	// Test simple function query
	goCode := `
package main

func HelloWorld() string {
	return "Hello, World!"
}

func privateFunc() int {
	return 42
}
`

	query := core.AgentQuery{
		Type: "function",
		Name: "HelloWorld",
	}

	result := provider.Query(goCode, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(result.Matches))
	}

	if result.Total != 1 {
		t.Errorf("Expected total 1, got %d", result.Total)
	}

	match := result.Matches[0]
	if match.Name != "HelloWorld" {
		t.Errorf("Expected match name 'HelloWorld', got '%s'", match.Name)
	}

	if match.Type != "function" {
		t.Errorf("Expected match type 'function', got '%s'", match.Type)
	}

	// Location should be set
	if match.Location.Line <= 0 {
		t.Error("Location line should be > 0")
	}

	// Content should contain function code
	if !strings.Contains(match.Content, "HelloWorld") {
		t.Error("Match content should contain function name")
	}
}

// TestQueryWithWildcard tests wildcard pattern matching
func TestQueryWithWildcard(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func TestFunc1() {}
func TestFunc2() {}
func OtherFunc() {}
`

	query := core.AgentQuery{
		Type: "function",
		Name: "Test*",
	}

	result := provider.Query(goCode, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Should match TestFunc1 and TestFunc2
	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}
}

func TestQueryFindsSecondaryExpandedName(t *testing.T) {
	provider := New(&multiBindingConfig{})
	source := `package main

var alpha, beta int
`

	result := provider.Query(source, core.AgentQuery{
		Type: "variable",
		Name: "beta",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 expanded secondary-name match, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "beta" {
		t.Fatalf("expected match name %q, got %q", "beta", result.Matches[0].Name)
	}
}

func TestQueryAppliesTypeSpecValidation(t *testing.T) {
	provider := New(&typeSpecValidatorConfig{})
	source := `package main

type Reader interface { Read() }
type User struct { Name string }
`

	result := provider.Query(source, core.AgentQuery{
		Type: "struct",
		Name: "*",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected only struct match after validation, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "User" {
		t.Fatalf("expected struct match %q, got %q", "User", result.Matches[0].Name)
	}
}

func TestQueryAppliesIfaceAliasValidation(t *testing.T) {
	provider := New(&ifaceAliasConfig{})
	source := `package main

type Reader interface { Read() }
type User struct { Name string }
`

	result := provider.Query(source, core.AgentQuery{
		Type: "iface",
		Name: "*",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected only interface match for iface alias, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "Reader" {
		t.Fatalf("expected interface match %q, got %q", "Reader", result.Matches[0].Name)
	}
}

func TestQueryAppliesVarAliasValidation(t *testing.T) {
	provider := New(&pythonVarAliasConfig{})
	source := "a = 1\nobj.x = 2\n"

	result := provider.Query(source, core.AgentQuery{
		Type: "var",
		Name: "*",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected only simple assignment match for var alias, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "a" {
		t.Fatalf("expected variable match %q, got %q", "a", result.Matches[0].Name)
	}
}

func TestQueryMatchesPhpExpandedPropertyNamesWithoutDollar(t *testing.T) {
	provider := New(&phpPropertyConfig{})
	source := "<?php class Test { public $foo, $bar; }"

	result := provider.Query(source, core.AgentQuery{
		Type: "property",
		Name: "foo",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected exact property match without dollar prefix, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "foo" {
		t.Fatalf("expected property match %q, got %q", "foo", result.Matches[0].Name)
	}
}

func TestQueryAppliesJavaScriptConstructorValidation(t *testing.T) {
	provider := New(&jsConstructorConfig{})
	source := `class Example {
	constructor() {}
	render() {}
}`

	result := provider.Query(source, core.AgentQuery{
		Type: "constructor",
		Name: "*",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected only constructor match, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "constructor" {
		t.Fatalf("expected constructor match %q, got %q", "constructor", result.Matches[0].Name)
	}
}

func TestQueryAppliesTypeScriptGetterValidation(t *testing.T) {
	provider := New(&tsSemanticMethodConfig{})
	source := `class Example {
	get value(): string { return this._value; }
	value(): string { return this._value; }
}`

	result := provider.Query(source, core.AgentQuery{
		Type: "getter",
		Name: "*",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected only getter match, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "value" {
		t.Fatalf("expected getter match %q, got %q", "value", result.Matches[0].Name)
	}
}

func TestQueryAppliesPhpConstructorValidation(t *testing.T) {
	provider := New(&phpConstructorConfig{})
	source := `<?php
class Example {
	public function __construct() {}
	public function render() {}
}`

	result := provider.Query(source, core.AgentQuery{
		Type: "ctor",
		Name: "*",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected only PHP constructor match, got %d", len(result.Matches))
	}
	if result.Matches[0].Name != "__construct" {
		t.Fatalf("expected constructor match %q, got %q", "__construct", result.Matches[0].Name)
	}
}

func TestQueryDedupesOverlappingExpandedJavaScriptVariables(t *testing.T) {
	provider := New(&jsOverlapVariableConfig{})
	source := "const foo = 1;"

	result := provider.Query(source, core.AgentQuery{
		Type: "variable",
		Name: "foo",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 deduped JS variable match, got %d", len(result.Matches))
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1 after dedupe, got %d", result.Total)
	}
	if result.Matches[0].Name != "foo" {
		t.Fatalf("expected JS variable match %q, got %q", "foo", result.Matches[0].Name)
	}
}

func TestTransformDedupesOverlappingExpandedPhpVariables(t *testing.T) {
	provider := New(&phpOverlapVariableConfig{})
	source := "<?php class Test { public $foo; }"

	result := provider.Transform(source, core.TransformOp{
		Method: "insert_before",
		Target: core.AgentQuery{
			Type: "variable",
			Name: "foo",
		},
		Content: "/*marker*/",
	})
	if result.Error != nil {
		t.Fatalf("transform failed: %v", result.Error)
	}

	if result.MatchCount != 1 {
		t.Fatalf("expected 1 deduped PHP variable target, got %d", result.MatchCount)
	}
	if strings.Count(result.Modified, "/*marker*/") != 1 {
		t.Fatalf("expected marker inserted once, got %q", result.Modified)
	}
}

func TestQueryDedupesOverlappingPhpAssignmentVariables(t *testing.T) {
	provider := New(&phpOverlapVariableConfig{})
	source := "<?php $foo = 1;"

	result := provider.Query(source, core.AgentQuery{
		Type: "variable",
		Name: "foo",
	})
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 deduped PHP assignment variable match, got %d", len(result.Matches))
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1 after PHP assignment dedupe, got %d", result.Total)
	}
	if result.Matches[0].Name != "foo" {
		t.Fatalf("expected PHP assignment variable match %q, got %q", "foo", result.Matches[0].Name)
	}
}

func TestTransformDedupesOverlappingPhpSimpleParameterVariables(t *testing.T) {
	provider := New(&phpOverlapVariableConfig{})
	source := "<?php function f($foo) {}"

	result := provider.Transform(source, core.TransformOp{
		Method: "insert_before",
		Target: core.AgentQuery{
			Type: "variable",
			Name: "foo",
		},
		Content: "/*marker*/",
	})
	if result.Error != nil {
		t.Fatalf("transform failed: %v", result.Error)
	}

	if result.MatchCount != 1 {
		t.Fatalf("expected 1 deduped PHP parameter variable target, got %d", result.MatchCount)
	}
	if strings.Count(result.Modified, "/*marker*/") != 1 {
		t.Fatalf("expected parameter marker inserted once, got %q", result.Modified)
	}
}

// TestQueryInvalidSource tests query with invalid source
func TestQueryInvalidSource(t *testing.T) {
	provider := newTestProvider()

	// Invalid Go source
	invalidCode := `
this is not valid go code {{{
`

	query := core.AgentQuery{
		Type: "function",
		Name: "test",
	}

	result := provider.Query(invalidCode, query)
	// Should still work but might not find anything meaningful
	if result.Error != nil {
		// Parse errors are not necessarily fatal for queries
		t.Logf("Query returned error (expected for invalid code): %v", result.Error)
	}
}

// TestQueryEmptySource tests query with empty source
func TestQueryEmptySource(t *testing.T) {
	provider := newTestProvider()

	query := core.AgentQuery{
		Type: "function",
		Name: "test",
	}

	result := provider.Query("", query)
	if result.Error != nil {
		t.Logf("Query error on empty source: %v", result.Error)
	}

	// Should return no matches for empty source
	if len(result.Matches) != 0 {
		t.Errorf("Expected 0 matches for empty source, got %d", len(result.Matches))
	}
}

// TestTransformReplace tests replace transformation
func TestTransformReplace(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func OldFunc() string {
	return "old"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "OldFunc",
		},
		Replacement: `func NewFunc() string {
	return "new"
}`,
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	if result.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", result.MatchCount)
	}

	// Should contain new function
	if !strings.Contains(result.Modified, "NewFunc") {
		t.Error("Modified code should contain 'NewFunc'")
	}

	// Should not contain old function
	if strings.Contains(result.Modified, "OldFunc") {
		t.Error("Modified code should not contain 'OldFunc'")
	}

	// Confidence should be reasonable
	if result.Confidence.Score <= 0 || result.Confidence.Score > 1 {
		t.Errorf("Invalid confidence score: %f", result.Confidence.Score)
	}

	if result.Confidence.Level == "" {
		t.Error("Confidence level should be set")
	}

	// Diff should be present
	if result.Diff == "" {
		t.Error("Diff should not be empty")
	}
}

func TestDoReplaceAppliesTargetsFromEnd(t *testing.T) {
	provider := newTestProvider()
	source := "abcdef"
	targets := []Target{
		{Match: Match{Name: "bc", StartByte: 1, EndByte: 3}},
		{Match: Match{Name: "ef", StartByte: 4, EndByte: 6}},
	}

	modified, err := provider.doReplace(source, targets, "X")
	if err != nil {
		t.Fatalf("replace failed: %v", err)
	}

	if modified != "aXdX" {
		t.Fatalf("expected end-to-start replacement result %q, got %q", "aXdX", modified)
	}
}

// TestTransformDelete tests delete transformation
func TestTransformDelete(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func KeepThis() string {
	return "keep"
}

func DeleteThis() string {
	return "delete"
}
`

	op := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "DeleteThis",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should not contain deleted function
	if strings.Contains(result.Modified, "DeleteThis") {
		t.Error("Modified code should not contain 'DeleteThis'")
	}

	// Should still contain kept function
	if !strings.Contains(result.Modified, "KeepThis") {
		t.Error("Modified code should still contain 'KeepThis'")
	}
}

// TestTransformInsertBefore tests insert before transformation
func TestTransformInsertBefore(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method:  "insert_before",
		Content: "// This comment is before the function\n",
		Target: core.AgentQuery{
			Type: "function",
			Name: "ExistingFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should contain the inserted comment
	if !strings.Contains(result.Modified, "// This comment is before the function") {
		t.Error("Modified code should contain inserted comment")
	}

	// Should still contain original function
	if !strings.Contains(result.Modified, "ExistingFunc") {
		t.Error("Modified code should still contain original function")
	}
}

// TestTransformInsertAfter tests insert after transformation
func TestTransformInsertAfter(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method:  "insert_after",
		Content: "\n// This comment is after the function",
		Target: core.AgentQuery{
			Type: "function",
			Name: "ExistingFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should contain the inserted comment
	if !strings.Contains(result.Modified, "// This comment is after the function") {
		t.Error("Modified code should contain inserted comment")
	}
}

// TestTransformAppend tests append transformation
func TestTransformAppend(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method: "append",
		Content: `
func AppendedFunc() string {
	return "appended"
}`,
		Target: core.AgentQuery{
			Type: "function",
			Name: "ExistingFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should contain both functions
	if !strings.Contains(result.Modified, "ExistingFunc") {
		t.Error("Modified code should contain original function")
	}

	if !strings.Contains(result.Modified, "AppendedFunc") {
		t.Error("Modified code should contain appended function")
	}
}

// TestTransformUnknownMethod tests unknown transformation method
func TestTransformUnknownMethod(t *testing.T) {
	provider := newTestProvider()

	// Create code with a function that will match, so we get past the "no matches" check
	goCode := `
package main

func test() string {
	return "test"
}
`

	op := core.TransformOp{
		Method: "unknown_method",
		Target: core.AgentQuery{
			Type: "function",
			Name: "test",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error == nil {
		t.Error("Transform should fail with unknown method")
	}

	if !strings.Contains(result.Error.Error(), "unknown transform method") {
		t.Errorf("Expected 'unknown transform method' error, got: %v", result.Error)
	}
}

// TestTransformNoMatches tests transformation with no target matches
func TestTransformNoMatches(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "NonExistentFunc",
		},
		Replacement: "func NewFunc() {}",
	}

	result := provider.Transform(goCode, op)
	if result.Error == nil {
		t.Error("Transform should fail when no matches found")
	}

	if !strings.Contains(result.Error.Error(), "no matches found") {
		t.Errorf("Expected 'no matches found' error, got: %v", result.Error)
	}
}

// TestTransformInvalidSource tests transformation with invalid source
func TestTransformInvalidSource(t *testing.T) {
	provider := newTestProvider()

	invalidCode := `this is not valid go {{{`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "test",
		},
		Replacement: "func Test() {}",
	}

	result := provider.Transform(invalidCode, op)
	if result.Error == nil {
		t.Error("Transform should fail with invalid source")
	}
}

// TestValidate tests source validation
func TestValidate(t *testing.T) {
	provider := newTestProvider()

	// Valid Go code
	validCode := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	result := provider.Validate(validCode)
	if !result.Valid {
		t.Errorf("Valid code should pass validation, errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for valid code, got: %v", result.Errors)
	}
}

// TestValidateInvalidCode tests validation with syntax errors
func TestValidateInvalidCode(t *testing.T) {
	provider := newTestProvider()

	// Invalid Go code with syntax error
	invalidCode := `
package main

func main() {
	fmt.Println("Hello, World!"  // Missing closing parenthesis
}
`

	result := provider.Validate(invalidCode)
	// Note: tree-sitter might not always detect all syntax errors
	// This is testing the error detection mechanism
	t.Logf("Validation result for invalid code - Valid: %t, Errors: %v",
		result.Valid, result.Errors)
}

// TestValidateEmptySource tests validation with empty source
func TestValidateEmptySource(t *testing.T) {
	provider := newTestProvider()

	result := provider.Validate("")
	// Empty source should be considered valid (though not useful)
	if !result.Valid {
		t.Error("Empty source should be valid")
	}
}

// TestMatchesPattern tests pattern matching utility
func TestMatchesPattern(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"exact_match", "TestFunc", true},
		{"wildcard_prefix", "Test*", true},
		{"wildcard_suffix", "*Func", true},
		{"wildcard_middle", "Test*Func", true},
		{"no_match", "OtherFunc", false},
		{"empty_pattern", "", true}, // Empty pattern matches all
		{"only_wildcard", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection or access the method somehow
			// Since matchesPattern is private, we'll test it through Query
			query := core.AgentQuery{
				Type: "function",
				Name: tt.pattern,
			}

			goCode := `
package main
func TestFunc() {}
`

			result := provider.Query(goCode, query)
			hasMatch := len(result.Matches) > 0

			if hasMatch != tt.expected {
				t.Errorf("Pattern '%s' expected %t, got %t", tt.pattern, tt.expected, hasMatch)
			}
		})
	}
}

// TestConfidenceCalculation tests confidence scoring
func TestConfidenceCalculation(t *testing.T) {
	provider := newTestProvider()

	// Test single target (should increase confidence)
	singleTargetCode := `
package main
func UniqueFunc() string {
	return "unique"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "UniqueFunc",
		},
		Replacement: "func NewFunc() {}",
	}

	result := provider.Transform(singleTargetCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Single target should have good confidence
	if result.Confidence.Score <= 0.8 {
		t.Errorf("Expected high confidence for single target, got %f", result.Confidence.Score)
	}
}

// TestConfidenceWithMultipleTargets tests confidence with many targets
func TestConfidenceWithMultipleTargets(t *testing.T) {
	provider := newTestProvider()

	// Code with many similar functions
	multipleTargetsCode := `
package main
func TestFunc1() {}
func TestFunc2() {}
func TestFunc3() {}
func TestFunc4() {}
func TestFunc5() {}
func TestFunc6() {}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "Test*", // Wildcard matches many
		},
		Replacement: "func NewFunc() {}",
	}

	result := provider.Transform(multipleTargetsCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Multiple targets should reduce confidence
	if result.Confidence.Score >= 0.8 {
		t.Errorf("Expected lower confidence for multiple targets, got %f", result.Confidence.Score)
	}

	// Should have multiple matches
	if result.MatchCount < 5 {
		t.Errorf("Expected at least 5 matches, got %d", result.MatchCount)
	}
}

// TestConfidenceWithDeleteOperation tests confidence for delete operations
func TestConfidenceWithDeleteOperation(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main
func TestFunc() string {
	return "test"
}
`

	op := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "TestFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Delete operations should have reduced confidence
	factors := result.Confidence.Factors
	foundDeleteFactor := false
	for _, factor := range factors {
		if factor.Name == "delete_operation" && factor.Impact < 0 {
			foundDeleteFactor = true
			break
		}
	}

	if !foundDeleteFactor {
		t.Error("Expected delete operation confidence factor")
	}
}

// TestDiffGeneration tests diff generation
func TestDiffGeneration(t *testing.T) {
	provider := newTestProvider()

	originalCode := `
package main
func OldFunc() {
	return "old"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "OldFunc",
		},
		Replacement: `func NewFunc() {
	return "new"
}`,
	}

	result := provider.Transform(originalCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Diff should contain changes
	if result.Diff == "" {
		t.Error("Diff should not be empty for different content")
	}

	if !strings.Contains(result.Diff, "NewFunc") {
		t.Error("Diff should contain new function name")
	}

	if !strings.Contains(result.Diff, "+") || !strings.Contains(result.Diff, "-") {
		t.Error("Diff should contain addition and deletion markers")
	}
}

// TestDiffGenerationNoChanges tests diff with identical content
func TestDiffGenerationNoChanges(t *testing.T) {
	provider := newTestProvider()

	// This is a bit contrived since we need a transformation that results in no change
	// We'll test the diff generator directly by mocking identical content
	goCode := `
package main
func TestFunc() {}
`

	// Try to replace with identical content (though this might still generate a diff
	// due to formatting differences)
	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "TestFunc",
		},
		Replacement: "func TestFunc() {}",
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// This test mainly ensures diff generation doesn't crash
	t.Logf("Diff result: '%s'", result.Diff)
}

// TestCacheBasicFunctionality tests basic cache operations
func TestCacheBasicFunctionality(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	source := []byte("package main\nfunc test() {}")

	// First call should be a miss
	tree1, hit1 := cache.GetOrParse(parser, source)
	if tree1 == nil {
		t.Fatal("GetOrParse returned nil tree")
	}

	if hit1 {
		t.Error("First call should be a miss")
	}

	// Second call should be a hit
	tree2, hit2 := cache.GetOrParse(parser, source)
	if tree2 == nil {
		t.Fatal("GetOrParse returned nil tree on second call")
	}

	if !hit2 {
		t.Error("Second call should be a hit")
	}

	// Clean up
	tree1.Close()
	tree2.Close()
}

// TestCacheStats tests cache statistics
func TestCacheStats(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	source1 := []byte("package main\nfunc test1() {}")
	source2 := []byte("package main\nfunc test2() {}")

	// Get initial stats for comparison (may not be zero due to other tests)
	initialStats := cache.Stats()
	initialHits := initialStats["hits"]
	initialMisses := initialStats["misses"]

	// First call - should be miss
	tree1, _ := cache.GetOrParse(parser, source1)
	tree1.Close()

	// Second call with same source - should be hit
	tree2, _ := cache.GetOrParse(parser, source1)
	tree2.Close()

	// Third call with different source - should be miss
	tree3, _ := cache.GetOrParse(parser, source2)
	tree3.Close()

	// Check final stats (compare to initial)
	stats := cache.Stats()
	hitsGain := stats["hits"] - initialHits
	missesGain := stats["misses"] - initialMisses

	// Should have gained at least 1 hit
	if hitsGain < 1 {
		t.Errorf("Expected at least 1 hit gain, got %d", hitsGain)
	}

	// Should have gained at least 2 misses (unless sources were already cached)
	if missesGain < 0 {
		t.Errorf("Expected non-negative misses gain, got %d", missesGain)
	}

	// Hit rate should be calculated
	if stats["hit_rate"] < 0 {
		t.Errorf("Expected non-negative hit rate, got %d", stats["hit_rate"])
	}
}

// TestCacheDifferentSources tests cache with different sources
func TestCacheDifferentSources(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	source1 := []byte("package main\nfunc unique_test1_func() {}")
	source2 := []byte("package main\nfunc unique_test2_func() {}")

	// Each should be a miss initially (using unique sources)
	tree1, hit1 := cache.GetOrParse(parser, source1)
	if hit1 {
		t.Log("First source was already cached (expected in shared cache)")
	}

	tree2, hit2 := cache.GetOrParse(parser, source2)
	if hit2 {
		t.Log("Second source was already cached (expected in shared cache)")
	}

	// Calling again with first source should be hit
	tree3, hit3 := cache.GetOrParse(parser, source1)
	if !hit3 {
		t.Error("Repeated first source should be a hit")
	}

	tree1.Close()
	tree2.Close()
	tree3.Close()
}

// TestCacheInvalidSource tests cache with invalid source
func TestCacheInvalidSource(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	invalidSource := []byte("unique_invalid_source_xyz {{{")

	// Should still work but parse invalid AST
	tree, hit := cache.GetOrParse(parser, invalidSource)
	if tree == nil {
		t.Fatal("GetOrParse should not return nil even for invalid source")
	}

	if hit {
		t.Log("Invalid source was already cached (expected in shared cache)")
	}

	// Second call should hit the cache
	tree2, hit2 := cache.GetOrParse(parser, invalidSource)
	if !hit2 {
		t.Error("Second call should be a hit even for invalid source")
	}

	tree.Close()
	tree2.Close()
}

// TestProviderQueryConcurrent ensures pooled parsers survive parallel use.
func TestProviderQueryConcurrent(t *testing.T) {
	provider := newTestProvider()
	source := `package main

func One() {}
func Two() {}
func Three() {}
`
	query := core.AgentQuery{Type: "function", Name: "*"}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				result := provider.Query(source, query)
				if result.Error != nil {
					t.Errorf("query error: %v", result.Error)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestProviderStats(t *testing.T) {
	provider := newTestProvider()
	code := `package main

func main() {}
`
	query := core.AgentQuery{Type: "function", Name: "main"}

	result := provider.Query(code, query)
	if result.Error != nil {
		t.Fatalf("query failed: %v", result.Error)
	}

	stats := provider.Stats()
	if stats.BorrowCount == 0 || stats.ReturnCount == 0 {
		t.Fatalf("expected borrow/return counters to be incremented, got %+v", stats)
	}
	if stats.Active != 0 {
		t.Fatalf("expected no active parsers after query, got %d", stats.Active)
	}
}

func TestMatchSeamKeepsParserNodesOutOfCore(t *testing.T) {
	if _, ok := reflect.TypeOf(core.CodeMatch{}).FieldByName("Node"); ok {
		t.Fatal("core.CodeMatch must not expose parser-native node state")
	}

	provider := newTestProvider()
	source := `package main

func demo() {}
`

	parser := sitter.NewParser()
	parser.SetLanguage(provider.config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	if tree == nil {
		t.Fatal("expected parsed tree")
	}
	defer tree.Close()

	targets := provider.findTargets(tree.RootNode(), source, core.AgentQuery{
		Type: "function",
		Name: "demo",
	})
	if len(targets) != 1 {
		t.Fatalf("expected 1 provider target, got %d", len(targets))
	}
	if targets[0].Node == nil {
		t.Fatal("provider-local targets must retain parser nodes for transforms")
	}
}

func TestExpandMatchesFallsBackForLegacyConfig(t *testing.T) {
	provider := New(&legacyMockConfig{
		language:   "go",
		extensions: []string{".go"},
	})
	source := `package main

func demo() {}
`

	parser := sitter.NewParser()
	parser.SetLanguage(provider.config.GetLanguage())
	tree := parser.Parse(nil, []byte(source))
	if tree == nil {
		t.Fatal("expected parsed tree")
	}
	defer tree.Close()

	targets := provider.findTargets(tree.RootNode(), source, core.AgentQuery{
		Type: "function",
		Name: "demo",
	})
	if len(targets) != 1 {
		t.Fatalf("expected 1 fallback target, got %d", len(targets))
	}
	if targets[0].Name != "demo" {
		t.Fatalf("expected fallback target name %q, got %q", "demo", targets[0].Name)
	}
	if targets[0].Node == nil {
		t.Fatal("fallback target must retain parser node")
	}
}

// TestCacheEmptySource tests cache with empty source
func TestCacheEmptySource(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	emptySource := []byte("")

	tree, hit := cache.GetOrParse(parser, emptySource)
	if tree == nil {
		t.Fatal("GetOrParse should not return nil for empty source")
	}

	if hit {
		t.Log("Empty source was already cached (expected in shared cache)")
	}

	tree.Close()
}
