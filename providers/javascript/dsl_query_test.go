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

func TestProviderQuerySupportsCapturePatternDSL(t *testing.T) {
	provider := New()
	source := `
function loadUser() {
	return api.fetch("/api/user");
}
`

	query, err := core.ParseDSL("call:$client.$method")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one captured call, got %d: %+v", result.Total, result.Matches)
	}
	captures := result.Matches[0].Captures
	if captures["client"] != "api" || captures["method"] != "fetch" {
		t.Fatalf("unexpected captures: %+v", captures)
	}
}

func TestProviderQuerySupportsCallArgumentAttributeDSL(t *testing.T) {
	provider := New()
	source := `
function loadUser() {
	return fetch("/api/user", { cache: "no-store" });
}

function loadPost() {
	return fetch("/api/post");
}
`

	query, err := core.ParseDSL(`call:fetch arg0="/api/user"`)
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one fetch call with matching first arg, got %d: %+v", result.Total, result.Matches)
	}
}

func TestProviderQuerySupportsDirectChildDSL(t *testing.T) {
	provider := New()
	source := `
class UserController {
	render() {
		return view();
	}
}
`

	query, err := core.ParseDSL("class:UserController >> method:render")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected direct method child match, got %d: %+v", result.Total, result.Matches)
	}
}

func TestProviderQuerySupportsSiblingOrderAttributes(t *testing.T) {
	provider := New()
	source := `
function first() {}
function second() {}
`

	query, err := core.ParseDSL("func:first before=func:second")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected first function before second, got %d: %+v", result.Total, result.Matches)
	}
}
