package matcher

import "testing"

func TestASTMatcher_RenameFunc(t *testing.T) {
	src := []byte(`package main\nfunc main() {}`)

	q := `((function_declaration name: (identifier) @id))`
	m, err := NewAST(q, "go")
	if err != nil {
		t.Fatalf("new ast: %v", err)
	}
	spans, err := m.Find(src)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	got := string(src[spans[0].Start:spans[0].End])
	if got != "main" {
		t.Fatalf("expected capture 'main', got %q", got)
	}
}
