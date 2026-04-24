package php

import (
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestProviderQuerySupportsMethodContainingCallDSL(t *testing.T) {
	provider := New()
	source := `<?php
class UserFormatter {
    public function format($name) {
        return strtoupper($name);
    }

    public function passthrough($name) {
        return $name;
    }
}
`

	query, err := core.ParseDSL("method:* > call:strtoupper")
	if err != nil {
		t.Fatalf("ParseDSL returned error: %v", err)
	}

	result := provider.Query(source, query)
	if result.Error != nil {
		t.Fatalf("Query returned error: %v", result.Error)
	}
	if result.Total != 1 {
		t.Fatalf("expected one method containing strtoupper, got %d: %+v", result.Total, result.Matches)
	}
	if result.Matches[0].Name != "format" {
		t.Fatalf("expected format match, got %+v", result.Matches[0])
	}
}

func TestProviderQuerySupportsReturnDSL(t *testing.T) {
	provider := New()
	source := `<?php
function token() {
    return "TOKEN";
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
