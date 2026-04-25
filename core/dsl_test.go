package core

import "testing"

func TestParseDSLParsesSimpleAliasAndWildcard(t *testing.T) {
	query, err := ParseDSL("func:Handle*")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Type != "func" {
		t.Fatalf("expected provider-owned func type, got %q", query.Type)
	}
	if query.Name != "Handle*" {
		t.Fatalf("expected Handle* name, got %q", query.Name)
	}
}

func TestParseDSLParsesHierarchyAndTypeAttribute(t *testing.T) {
	query, err := ParseDSL("struct:* > field:Secret string")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Type != "struct" || query.Name != "*" {
		t.Fatalf("unexpected parent query: %+v", query)
	}
	if query.Contains == nil {
		t.Fatal("expected child query")
	}
	if query.Contains.Type != "field" || query.Contains.Name != "Secret" {
		t.Fatalf("unexpected child query: %+v", query.Contains)
	}
	if query.Contains.Attributes["type"] != "string" {
		t.Fatalf("expected child type attribute string, got %+v", query.Contains.Attributes)
	}
}

func TestParseDSLParsesNegationAndLogicalOperators(t *testing.T) {
	negated, err := ParseDSL("!func:Test*")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}
	if negated.Operator != "NOT" || len(negated.Operands) != 1 {
		t.Fatalf("expected NOT query with one operand, got %+v", negated)
	}
	if negated.Operands[0].Type != "func" || negated.Operands[0].Name != "Test*" {
		t.Fatalf("unexpected negated operand: %+v", negated.Operands[0])
	}

	logical, err := ParseDSL("func:* | method:*")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}
	if logical.Operator != "OR" || len(logical.Operands) != 2 {
		t.Fatalf("expected OR query with two operands, got %+v", logical)
	}
}

func TestParseDSLPreservesProviderSpecificKind(t *testing.T) {
	query, err := ParseDSL("def:load_user")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Type != "def" {
		t.Fatalf("expected raw provider-specific kind, got %q", query.Type)
	}
}

func TestParseDSLHonorsParenthesesAndPrecedence(t *testing.T) {
	query, err := ParseDSL("(func:* | method:*) > call:os.Getenv")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Operator != "OR" || len(query.Operands) != 2 {
		t.Fatalf("expected OR parent query, got %+v", query)
	}
	for _, operand := range query.Operands {
		if operand.Contains == nil {
			t.Fatalf("expected contains child on operand %+v", operand)
		}
		if operand.Contains.Type != "call" || operand.Contains.Name != "os.Getenv" {
			t.Fatalf("unexpected contains child: %+v", operand.Contains)
		}
	}
}

func TestParseDSLParsesKeyValueAttributes(t *testing.T) {
	query, err := ParseDSL("field:Secret type=string visibility=private")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Attributes["type"] != "string" {
		t.Fatalf("expected type=string, got %+v", query.Attributes)
	}
	if query.Attributes["visibility"] != "private" {
		t.Fatalf("expected visibility=private, got %+v", query.Attributes)
	}
}

func TestParseDSLParsesDirectChildOperator(t *testing.T) {
	query, err := ParseDSL("class:User >> method:render")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Contains == nil {
		t.Fatal("expected child query")
	}
	if !query.ContainsDirect {
		t.Fatalf("expected direct child containment, got %+v", query)
	}
	if query.Contains.Type != "method" || query.Contains.Name != "render" {
		t.Fatalf("unexpected child query: %+v", query.Contains)
	}
}

func TestParseDSLParsesCapturePatterns(t *testing.T) {
	query, err := ParseDSL("call:$pkg.$name")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	if query.Name != "$pkg.$name" {
		t.Fatalf("expected capture pattern to be preserved, got %q", query.Name)
	}
}

func TestParseDSLRejectsMalformedExpressions(t *testing.T) {
	cases := []string{
		"func:* >",
		"func:* >>",
		"(func:* | method:*",
		"func:* | | method:*",
		"field:Secret type=",
		":MissingKind",
	}
	for _, dsl := range cases {
		t.Run(dsl, func(t *testing.T) {
			if _, err := ParseDSL(dsl); err == nil {
				t.Fatalf("expected ParseDSL(%q) to fail", dsl)
			}
		})
	}
}
