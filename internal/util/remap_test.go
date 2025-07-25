package util

import (
	"regexp"
	"testing"
)

func TestRemapAllMatches_Simple(t *testing.T) {
	orig := "foo   bar"
	norm, n2o, o2n := NormalizeWhitespace(orig) // "foo bar"
	re := regexp.MustCompile(`bar`)
	matchesNorm := re.FindAllStringSubmatchIndex(norm, -1)
	if len(matchesNorm) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchesNorm))
	}

	matchesOrig := RemapAllMatches(matchesNorm, n2o, o2n)
	if len(matchesOrig) != 1 {
		t.Fatalf("expected 1 remapped match, got %d", len(matchesOrig))
	}
	start, end := matchesOrig[0][0], matchesOrig[0][1]
	if orig[start:end] != "bar" {
		t.Fatalf("unexpected slice: %q", orig[start:end])
	}
}

func TestRemapAllMatches_StartOnCollapsedSpace(t *testing.T) {
	orig := "foo \t \nbar"
	norm, n2o, o2n := NormalizeWhitespace(orig) // "foo bar"
	// Regex que matchea " foo bar" desde el espacio
	re := regexp.MustCompile(`\sbar`)
	matchesNorm := re.FindAllStringSubmatchIndex(norm, -1)
	if len(matchesNorm) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchesNorm))
	}
	// start cae sobre el espacio colapsado
	if matchesNorm[0][0] != 3 {
		t.Fatalf("expected start 3 in normalized, got %d", matchesNorm[0][0])
	}

	matchesOrig := RemapAllMatches(matchesNorm, n2o, o2n)
	start, end := matchesOrig[0][0], matchesOrig[0][1]
	// El remapeo debe apuntar a algún rango que incluya " bar" en el original
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("invalid remapped span: [%d,%d)", start, end)
	}
	if orig[end-3:end] != "bar" { // sanity quick check
		t.Fatalf("expected to end in 'bar', got %q", orig[end-3:end])
	}
}

func TestRemapAllMatches_WithSubmatches(t *testing.T) {
	orig := "func  A()  {"
	norm, n2o, o2n := NormalizeWhitespace(orig) // "func A() {"
	re := regexp.MustCompile(`func\s+(\w+)\(([^)]*)\)`)
	matchesNorm := re.FindAllStringSubmatchIndex(norm, -1)
	if len(matchesNorm) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchesNorm))
	}
	// 0: full, 1: name, 2: args
	if len(matchesNorm[0]) < 6 {
		t.Fatalf("expected match + 2 groups, got %v", matchesNorm[0])
	}

	matchesOrig := RemapAllMatches(matchesNorm, n2o, o2n)
	rm := matchesOrig[0]
	if len(rm) != len(matchesNorm[0]) {
		t.Fatalf("len mismatch: got %d want %d", len(rm), len(matchesNorm[0]))
	}

	// Extra sanity: grupo 1 debe mapear a "A"
	g1s, g1e := rm[2], rm[3]
	if orig[g1s:g1e] != "A" {
		t.Fatalf("expected group1 to map to 'A', got %q", orig[g1s:g1e])
	}
}
