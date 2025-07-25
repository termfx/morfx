package core

import (
	"regexp"
	"testing"

	"github.com/garaekz/fileman/internal/model"
)

func TestManipulatorApply_NormalizeWhitespace_ReplaceSignature(t *testing.T) {
	content := "func  A()  {\r\n\t// TODO: add error handling\r\n\tfmt.Println(\"hi\")\r\n}\r\n"

	// Regla tipo r001 (pero activando NormalizeWhitespace)
	cfg := model.ModificationConfig{
		RuleID:              "r001-add-error-return",
		Pattern:             `func (\w+)\(([^)]*)\)`,
		Replacement:         `func $1($2) error`,
		Operation:           model.OpReplace,
		Occurrences:         "all",
		NormalizeWhitespace: true,
		Multiline:           true,
		DotAll:              false,
		Context: &model.Context{
			Before:       `// TODO: add error handling`,
			After:        `{`,
			WindowBefore: 5,
			WindowAfter:  2,
		},
	}

	m := NewManipulator(cfg)

	out, changes, err := m.Apply(content)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}

	// Debe haber insertado " error" en la firma, preservando CRLF e indentación/resto
	re := regexp.MustCompile(`func\s+A\(\)\s+error\s+\{\r\n`)
	if !re.MatchString(out) {
		t.Fatalf("result doesn't contain expected signature with error:\n%s", out)
	}

	// Verifica que las líneas reportadas por Change tengan sentido
	ch := changes[0]
	if ch.LineStart <= 0 || ch.LineEnd < ch.LineStart {
		t.Fatalf("invalid line range in change: %+v", ch)
	}
}

func TestManipulatorApply_NormalizeWhitespace_FirstOccurrence(t *testing.T) {
	content := "func Foo() {}\nfunc   Bar (  ) {}\n"
	cfg := model.ModificationConfig{
		RuleID:              "r002-first",
		Pattern:             `func (\w+)\(([^)]*)\)`,
		Replacement:         `func $1($2) error`,
		Operation:           model.OpReplace,
		Occurrences:         "first",
		NormalizeWhitespace: true,
		Multiline:           true,
	}
	m := NewManipulator(cfg)

	out, changes, err := m.Apply(content)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	// La primera firma modificada debe ser Foo, no Bar
	if !regexp.MustCompile(`func Foo\(\) error`).MatchString(out) {
		t.Fatalf("Foo signature wasn't modified:\n%s", out)
	}
	if regexp.MustCompile(`func\s+Bar\s*\(\s*\)\s*error`).MatchString(out) {
		t.Fatalf("Bar should not have been modified with occurrences=first:\n%s", out)
	}
}

func TestManipulatorApply_NoNormalize_Regression(t *testing.T) {
	content := "func A() {}\n"
	cfg := model.ModificationConfig{
		RuleID:              "r003-no-norm",
		Pattern:             `func (\w+)\(([^)]*)\)`,
		Replacement:         `func $1($2) error`,
		Operation:           model.OpReplace,
		Occurrences:         "all",
		NormalizeWhitespace: false,
		Multiline:           true,
	}
	m := NewManipulator(cfg)

	out, changes, err := m.Apply(content)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if out != "func A() error {}\n" {
		t.Fatalf("unexpected output: %q", out)
	}
}
