package engine

import (
	"path/filepath"
	"testing"
)

func TestRootPolicyRejectsOutsideRoots(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	p := newRootPolicy([]string{root})

	outside := filepath.Join(filepath.Dir(root), "x.go")
	if _, err := p.ValidatePath(outside); err == nil {
		t.Fatalf("expected outside-root path to be rejected")
	}
}
