package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

const providerTemplate = `package {{.Package}}

import (
	"fmt"
	"strings"

	"github.com/garaekz/fileman/internal/provider"
	"github.com/garaekz/fileman/internal/types"
	sitter "github.com/smacker/go-tree-sitter"
	{{.Package}}_sitter "github.com/smacker/go-tree-sitter/{{.SitterPackage}}"
)

// {{.Name}}Provider implements the LanguageProvider interface for {{.DisplayName}} language support
type {{.Name}}Provider struct {
	provider.BaseProvider
}

// NewProvider creates a new instance of the {{.DisplayName}} language provider
func NewProvider() provider.LanguageProvider {
	p := &{{.Name}}Provider{}
	p.Initialize()
	return p
}

// Initialize sets up the {{.DisplayName}} provider with language-specific mappings
func (p *{{.Name}}Provider) Initialize() {
	// Define how universal kinds map to {{.DisplayName}} AST
	mappings := []provider.NodeMapping{
		// TODO: Define your language mappings here
		// Example mappings - adjust based on your language's AST structure
		{
			Kind:        "function",
			NodeTypes:   []string{"{{.FunctionNode}}"},
			NameCapture: "@name",
			Template:    {{.BackQuote}}({{.FunctionNode}} name: (identifier) @name %s){{.BackQuote}},
		},
		{
			Kind:        "class",
			NodeTypes:   []string{"{{.ClassNode}}"},
			NameCapture: "@name",
			Template:    {{.BackQuote}}({{.ClassNode}} name: (identifier) @name %s){{.BackQuote}},
		},
		{
			Kind:        "variable",
			NodeTypes:   []string{"{{.VariableNode}}"},
			NameCapture: "@name",
			Template:    {{.BackQuote}}({{.VariableNode}} name: (identifier) @name %s){{.BackQuote}},
		},
		// Add more mappings for:
		// - method
		// - import
		// - constant
		// - field
		// - call
		// - assignment
		// - condition
		// - loop
		// - block
		// - comment
		// - decorator (if applicable)
		// - type
	}

	p.BuildMappings(mappings)
}

// Lang returns the canonical name of the language
func (p *{{.Name}}Provider) Lang() string {
	return "{{.Lang}}"
}

// Aliases returns alternative names for this language
func (p *{{.Name}}Provider) Aliases() []string {
	return []string{ {{range .Aliases}}"{{.}}", {{end}} }
}

// Extensions returns file extensions for this language
func (p *{{.Name}}Provider) Extensions() []string {
	return []string{ {{range .Extensions}}"{{.}}", {{end}} }
}

// GetSitterLanguage returns the Tree-sitter language for {{.DisplayName}}
func (p *{{.Name}}Provider) GetSitterLanguage() *sitter.Language {
	return {{.Package}}_sitter.GetLanguage()
}

// TranslateQuery translates a universal query to {{.DisplayName}}-specific Tree-sitter query
func (p *{{.Name}}Provider) TranslateQuery(q *provider.Query) (string, error) {
	mappings := p.TranslateKind(q.Kind)
	if len(mappings) == 0 {
		return "", fmt.Errorf("unsupported node kind: %s", q.Kind)
	}

	// Build query based on pattern and attributes
	var queries []string
	for _, mapping := range mappings {
		query := p.buildQueryFromMapping(mapping, q)
		if query != "" {
			queries = append(queries, query)
		}
	}

	if len(queries) == 0 {
		return "", fmt.Errorf("no valid queries generated for kind: %s", q.Kind)
	}

	return strings.Join(queries, "\n"), nil
}

// buildQueryFromMapping constructs a Tree-sitter query from a mapping and query
func (p *{{.Name}}Provider) buildQueryFromMapping(mapping provider.NodeMapping, q *provider.Query) string {
	// Handle pattern matching
	patternConstraint := ""
	if q.Pattern != "" && q.Pattern != "*" {
		regexPattern := p.convertWildcardToRegex(q.Pattern)
		patternConstraint = fmt.Sprintf({{.BackQuote}}(#match? %s "%s"){{.BackQuote}}, mapping.NameCapture, regexPattern)
	}

	// Handle type constraints
	typeConstraint := ""
	if typeAttr, hasType := q.Attributes["type"]; hasType && mapping.TypeCapture != "" {
		typeRegex := p.convertWildcardToRegex(typeAttr)
		typeConstraint = fmt.Sprintf({{.BackQuote}}(#match? %s "%s"){{.BackQuote}}, mapping.TypeCapture, typeRegex)
	}

	// Combine constraints
	constraints := []string{}
	if patternConstraint != "" {
		constraints = append(constraints, patternConstraint)
	}
	if typeConstraint != "" {
		constraints = append(constraints, typeConstraint)
	}

	constraintStr := strings.Join(constraints, " ")
	
	// Special handling for templates with multiple %s placeholders
	if strings.Count(mapping.Template, "%s") > 1 {
		placeholders := make([]interface{}, strings.Count(mapping.Template, "%s"))
		for i := range placeholders {
			placeholders[i] = constraintStr
		}
		return fmt.Sprintf(mapping.Template, placeholders...)
	}
	
	return fmt.Sprintf(mapping.Template, constraintStr)
}

// convertWildcardToRegex converts wildcard patterns to regex
func (p *{{.Name}}Provider) convertWildcardToRegex(pattern string) string {
	// Escape regex special characters except * and ?
	escaped := strings.ReplaceAll(pattern, ".", "\\.")
	escaped = strings.ReplaceAll(escaped, "+", "\\+")
	escaped = strings.ReplaceAll(escaped, "^", "\\^")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "(", "\\(")
	escaped = strings.ReplaceAll(escaped, ")", "\\)")
	escaped = strings.ReplaceAll(escaped, "[", "\\[")
	escaped = strings.ReplaceAll(escaped, "]", "\\]")
	escaped = strings.ReplaceAll(escaped, "{", "\\{")
	escaped = strings.ReplaceAll(escaped, "}", "\\}")
	escaped = strings.ReplaceAll(escaped, "|", "\\|")

	// Convert wildcards to regex
	escaped = strings.ReplaceAll(escaped, "*", ".*")
	escaped = strings.ReplaceAll(escaped, "?", ".")

	// Anchor the pattern
	return "^" + escaped + "$"
}

// GetNodeKind determines the universal kind of a Tree-sitter node
func (p *{{.Name}}Provider) GetNodeKind(node *sitter.Node) provider.NodeKind {
	// TODO: Implement based on your language's AST structure
	switch node.Type() {
	case "{{.FunctionNode}}":
		return "function"
	case "{{.ClassNode}}":
		return "class"
	case "{{.VariableNode}}":
		return "variable"
	// Add more cases...
	default:
		return provider.NodeKind(node.Type()) // Fallback to node type
	}
}

// GetNodeName extracts the name/identifier from a Tree-sitter node
func (p *{{.Name}}Provider) GetNodeName(node *sitter.Node, source []byte) string {
	// TODO: Implement based on your language's AST structure
	switch node.Type() {
	case "{{.FunctionNode}}", "{{.ClassNode}}":
		return p.extractIdentifier(node, "name", source)
	case "{{.VariableNode}}":
		return p.extractIdentifier(node, "name", source)
	// Add more cases...
	default:
		return node.Content(source)
	}
}

// Helper method for name extraction
func (p *{{.Name}}Provider) extractIdentifier(node *sitter.Node, fieldName string, source []byte) string {
	if child := node.ChildByFieldName(fieldName); child != nil {
		return child.Content(source)
	}
	return ""
}

// GetNodeScope provides {{.DisplayName}}-specific scope detection
func (p *{{.Name}}Provider) GetNodeScope(node *sitter.Node) provider.ScopeType {
	// TODO: Implement based on your language's scope rules
	switch node.Type() {
	case "program", "source_file", "module":
		return "file"
	case "{{.ClassNode}}":
		return "class"
	case "{{.FunctionNode}}":
		return "function"
	case "block", "compound_statement":
		return "block"
	default:
		if node.Parent() != nil {
			return p.GetNodeScope(node.Parent())
		}
		return "file"
	}
}
`

func main() {
	var lang, displayName, sitterPkg string
	var aliases, extensions string

	flag.StringVar(&lang, "lang", "", "Language name (e.g., ruby)")
	flag.StringVar(&displayName, "display", "", "Display name (e.g., Ruby)")
	flag.StringVar(&sitterPkg, "sitter", "", "Tree-sitter package name (defaults to lang)")
	flag.StringVar(&aliases, "aliases", "", "Comma-separated aliases")
	flag.StringVar(&extensions, "ext", "", "Comma-separated file extensions")
	flag.Parse()

	if lang == "" {
		fmt.Println("Usage: morfx-provider-gen -lang=<language> [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  -lang string     Language name (required, e.g., 'ruby')")
		fmt.Println("  -display string  Display name (e.g., 'Ruby')")
		fmt.Println("  -sitter string   Tree-sitter package name (defaults to lang)")
		fmt.Println("  -aliases string  Comma-separated aliases (e.g., 'rb,ruby')")
		fmt.Println("  -ext string      Comma-separated extensions (e.g., '.rb,.erb')")
		fmt.Println("\nExample:")
		fmt.Println("  morfx-provider-gen -lang=ruby -display=Ruby -aliases=ruby,rb -ext=.rb,.erb")
		os.Exit(1)
	}

	// Set defaults
	if displayName == "" {
		displayName = strings.Title(lang)
	}
	if sitterPkg == "" {
		sitterPkg = lang
	}

	// Parse aliases and extensions
	var aliasesList []string
	if aliases != "" {
		aliasesList = strings.Split(aliases, ",")
		for i := range aliasesList {
			aliasesList[i] = strings.TrimSpace(aliasesList[i])
		}
	} else {
		aliasesList = []string{lang}
	}

	var extensionsList []string
	if extensions != "" {
		extensionsList = strings.Split(extensions, ",")
		for i := range extensionsList {
			ext := strings.TrimSpace(extensionsList[i])
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			extensionsList[i] = ext
		}
	} else {
		extensionsList = []string{fmt.Sprintf(".%s", lang)}
	}

	// Guess common AST node names (can be customized per language)
	functionNode := "function_definition"
	classNode := "class_definition"
	variableNode := "variable_declaration"

	// Language-specific adjustments
	switch lang {
	case "go", "golang":
		functionNode = "function_declaration"
		classNode = "type_declaration"
		variableNode = "var_declaration"
	case "javascript", "js", "typescript", "ts":
		functionNode = "function_declaration"
		classNode = "class_declaration"
		variableNode = "variable_declarator"
	case "python", "py":
		functionNode = "function_definition"
		classNode = "class_definition"
		variableNode = "assignment"
	case "ruby", "rb":
		functionNode = "method"
		classNode = "class"
		variableNode = "assignment"
	case "java":
		functionNode = "method_declaration"
		classNode = "class_declaration"
		variableNode = "variable_declarator"
	case "c", "cpp", "c++":
		functionNode = "function_definition"
		classNode = "class_specifier"
		variableNode = "declaration"
	case "rust", "rs":
		functionNode = "function_item"
		classNode = "struct_item"
		variableNode = "let_declaration"
	case "php":
		functionNode = "function_definition"
		classNode = "class_declaration"
		variableNode = "simple_parameter"
	}

	// Generate provider skeleton
	data := map[string]any{
		"Package":       lang,
		"SitterPackage": sitterPkg,
		"Name":          strings.Title(lang),
		"DisplayName":   displayName,
		"Lang":          lang,
		"Aliases":       aliasesList,
		"Extensions":    extensionsList,
		"FunctionNode":  functionNode,
		"ClassNode":     classNode,
		"VariableNode":  variableNode,
		"BackQuote":     "`",
	}

	tmpl, err := template.New("provider").Parse(providerTemplate)
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Create directory
	providerDir := filepath.Join("internal", "lang", lang)
	if err := os.MkdirAll(providerDir, 0o755); err != nil {
		fmt.Printf("Error creating directory %s: %v\n", providerDir, err)
		os.Exit(1)
	}

	// Write provider file
	providerFile := filepath.Join(providerDir, "provider.go")
	file, err := os.Create(providerFile)
	if err != nil {
		fmt.Printf("Error creating file %s: %v\n", providerFile, err)
		os.Exit(1)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		fmt.Printf("Error executing template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Provider skeleton created at %s\n", providerFile)
	fmt.Println("\n📝 Next steps:")
	fmt.Printf("1. Install Tree-sitter grammar: go get github.com/smacker/go-tree-sitter/%s\n", sitterPkg)
	fmt.Println("2. Review and adjust the NodeMappings in Initialize()")
	fmt.Println("3. Implement GetNodeKind() and GetNodeName() methods")
	fmt.Println("4. Test your provider: go test ./internal/lang/" + lang)
	fmt.Println("5. Register in AutoRegister() function in internal/registry/registry.go")
	fmt.Println("\n💡 Tips:")
	fmt.Println("- Use tree-sitter playground to explore AST structure: https://tree-sitter.github.io/tree-sitter/playground")
	fmt.Println("- Check existing providers (Go, Python, JS, TS) for reference")
	fmt.Println("- The template provides a working base - customize mappings for your language's AST")
}
