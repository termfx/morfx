package util

import (
	"bytes"
	"testing"
)

func TestNormalizeWhitespace_Empty(t *testing.T) {
	n, n2o, o2n := NormalizeWhitespace("")
	if n != "" {
		t.Fatalf("expected empty normalized, got %q", n)
	}
	if len(n2o) != 0 || len(o2n) != 0 {
		t.Fatalf("expected empty maps, got n2o=%v o2n=%v", n2o, o2n)
	}
}

func TestNormalizeWhitespace_OnlyWhitespace(t *testing.T) {
	s := " \t \n  "
	n, n2o, o2n := NormalizeWhitespace(s)
	if n != "" {
		t.Fatalf("expected empty normalized, got %q", n)
	}
	if len(n2o) != 0 {
		t.Fatalf("expected n2o empty, got %v", n2o)
	}
	for i, v := range o2n {
		if v != -1 {
			t.Fatalf("expected o2n[%d] == -1, got %d", i, v)
		}
	}
}

func TestNormalizeWhitespace_UTF8Multibyte(t *testing.T) {
	s := "á  β\t中  y"
	n, n2o, o2n := NormalizeWhitespace(s)
	if n != "á β 中 y" {
		t.Fatalf("unexpected normalized: %q", n)
	}
	// Basic sanity: lengths
	if len(n2o) != len(n) {
		t.Fatalf("n2o len mismatch: got %d, want %d", len(n2o), len(n))
	}
	if len(o2n) != len(s) {
		t.Fatalf("o2n len mismatch: got %d, want %d", len(o2n), len(s))
	}
	// Ensure every emitted byte in normalized maps to a valid original byte
	for i, v := range n2o {
		if v < 0 || v >= len(s) {
			t.Fatalf("n2o[%d]=%d out of bounds (len(s)=%d)", i, v, len(s))
		}
	}
}

func TestNormalizeWhitespace_CRLF(t *testing.T) {
	s := "\r\na\r\nb\r\n"
	n, _, _ := NormalizeWhitespace(s)
	if n != "a b" {
		t.Fatalf("unexpected normalized: %q", n)
	}
}

func TestNormalizeWhitespace_CollapsedSpaceMapsToFirstOriginalByte(t *testing.T) {
	// "foo<ws>bar" -> "foo bar"
	s := "foo \t  \nbar"
	n, n2o, _ := NormalizeWhitespace(s)
	if n != "foo bar" {
		t.Fatalf("unexpected normalized: %q", n)
	}
	// Find the single collapsed space in normalized
	spaceIdx := bytes.IndexByte([]byte(n), ' ')
	if spaceIdx == -1 {
		t.Fatalf("no space in normalized output")
	}
	origIdx := n2o[spaceIdx]
	// In original, first whitespace byte is at index 3 ("foo| …")
	if origIdx != 3 {
		t.Fatalf("collapsed space should map to first original ws byte 3, got %d", origIdx)
	}
}
