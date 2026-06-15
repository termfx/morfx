package engine

import "testing"

func TestNewBuildsEngineWithDefaultProviders(t *testing.T) {
	e, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if e == nil {
		t.Fatalf("expected engine instance")
	}
	if e.runtime == nil {
		t.Fatalf("expected runtime")
	}
	if e.runtime.Providers == nil {
		t.Fatalf("expected provider registry")
	}
}
