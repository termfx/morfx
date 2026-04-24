package python

import (
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestProviderOwnsDefDSLKeyword(t *testing.T) {
	provider := New()
	source := `def load_user():
    return "ok"

class User:
    pass
`

	query, err := core.ParseDSL("def:load_*")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one Python def match, got %d: %+v", result.Total, result.Matches)
	}
	if result.Matches[0].Name != "load_user" {
		t.Fatalf("expected load_user match, got %+v", result.Matches[0])
	}
	if result.Matches[0].Type != "function" {
		t.Fatalf("expected provider-normalized function type, got %+v", result.Matches[0])
	}
}

func TestProviderQuerySupportsFunctionContainingCallDSL(t *testing.T) {
	provider := New()
	source := `import os

def load_token():
    return os.getenv("TOKEN")

def local_token():
    return "TOKEN"
`

	query, err := core.ParseDSL("def:* > call:os.getenv")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one Python def containing os.getenv, got %d: %+v", result.Total, result.Matches)
	}
	if result.Matches[0].Name != "load_token" {
		t.Fatalf("expected load_token match, got %+v", result.Matches[0])
	}
}

func TestProviderQuerySupportsReturnDSL(t *testing.T) {
	provider := New()
	source := `def load_token():
    return "TOKEN"
`

	query, err := core.ParseDSL("return:*")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one return match, got %d: %+v", result.Total, result.Matches)
	}
}
