package javascript

import (
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestProviderQuerySupportsFunctionContainingCallDSL(t *testing.T) {
	provider := New()
	source := `
function loadUser() {
	return fetch("/api/user");
}

function localUser() {
	return { name: "Ada" };
}
`

	query, err := core.ParseDSL("func:* > call:fetch")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one function containing fetch, got %d: %+v", result.Total, result.Matches)
	}
	if result.Matches[0].Name != "loadUser" {
		t.Fatalf("expected loadUser match, got %+v", result.Matches[0])
	}
}

func TestProviderQuerySupportsReturnDSL(t *testing.T) {
	provider := New()
	source := `
function loadUser() {
	return fetch("/api/user");
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
