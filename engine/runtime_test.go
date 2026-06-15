package engine

import "testing"

func TestBuildRuntimeRegistersBuiltins(t *testing.T) {
	t.Parallel()

	rt, err := BuildRuntime(Config{})
	if err != nil {
		t.Fatalf("BuildRuntime() error = %v", err)
	}
	if _, ok := rt.Providers.Get("go"); !ok {
		t.Fatalf("expected go provider")
	}
	if _, ok := rt.Providers.Get("python"); !ok {
		t.Fatalf("expected python provider")
	}
}
