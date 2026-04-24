package golang

import (
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestProviderQuerySupportsDSLStyleHierarchy(t *testing.T) {
	provider := New()
	source := `package main

import "os"

func Load() string {
	return os.Getenv("TOKEN")
}

func Ignore() string {
	return "TOKEN"
}
`

	query, err := core.ParseDSL("func:* > call:os.Getenv")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one function containing os.Getenv, got %d: %+v", result.Total, result.Matches)
	}
	if result.Matches[0].Name != "Load" {
		t.Fatalf("expected Load match, got %+v", result.Matches[0])
	}
	if result.Matches[0].Type != "function" {
		t.Fatalf("expected provider-normalized function type, got %+v", result.Matches[0])
	}
}

func TestProviderQuerySupportsDSLStyleFieldTypeConstraint(t *testing.T) {
	provider := New()
	source := `package main

type User struct {
	Secret string
	Count int
}

type Other struct {
	Secret int
}
`

	query, err := core.ParseDSL("struct:* > field:Secret string")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one struct with Secret string, got %d: %+v", result.Total, result.Matches)
	}
	if result.Matches[0].Name != "User" {
		t.Fatalf("expected User match, got %+v", result.Matches[0])
	}
}

func TestProviderDoesNotTreatPythonDefAsGoFunctionDSL(t *testing.T) {
	provider := New()
	source := `package main

func Load() string {
	return "ok"
}
`

	query, err := core.ParseDSL("def:Load")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 0 {
		t.Fatalf("expected Go provider not to translate def, got %d: %+v", result.Total, result.Matches)
	}
}

func TestProviderQuerySupportsReturnDSL(t *testing.T) {
	provider := New()
	source := `package main

func Load() string {
	return "ok"
}
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
