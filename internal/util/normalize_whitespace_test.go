package util

import (
	"bytes"
	"reflect"
	"testing"
	"unicode/utf8"
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
	s := " 	 \n  "
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

func TestNormalizeWhitespace_Extended(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedNormalized string
		expectedN2O        []int // Simplified check for key mappings
		expectedO2N        []int // Simplified check for key mappings
	}{
		{
			name:               "No whitespace",
			input:              "hello",
			expectedNormalized: "hello",
			expectedN2O:        []int{0, 1, 2, 3, 4},
			expectedO2N:        []int{0, 1, 2, 3, 4},
		},
		{
			name:               "Leading whitespace",
			input:              "  hello",
			expectedNormalized: "hello",
			expectedN2O:        []int{2, 3, 4, 5, 6},
			expectedO2N:        []int{-1, -1, 0, 1, 2, 3, 4},
		},
		{
			name:               "Trailing whitespace",
			input:              "hello  ",
			expectedNormalized: "hello",
			expectedN2O:        []int{0, 1, 2, 3, 4},
			expectedO2N:        []int{0, 1, 2, 3, 4, -1, -1},
		},
		{
			name:               "Leading and trailing whitespace",
			input:              "  hello  ",
			expectedNormalized: "hello",
			expectedN2O:        []int{2, 3, 4, 5, 6},
			expectedO2N:        []int{-1, -1, 0, 1, 2, 3, 4, -1, -1},
		},
		{
			name:               "Multiple internal whitespace sequences",
			input:              "hello   world  again",
			expectedNormalized: "hello world again",
			expectedN2O:        []int{0, 1, 2, 3, 4, 5, 8, 9, 10, 11, 12, 14, 15, 16, 17},
			expectedO2N:        []int{0, 1, 2, 3, 4, 5, -1, -1, 6, 7, 8, 9, 10, -1, 11, 12, 13, 14},
		},
		{
			name:               "Mixed whitespace characters",
			input:              "hello\t\n world",
			expectedNormalized: "hello world",
			expectedN2O:        []int{0, 1, 2, 3, 4, 5, 8, 9, 10, 11},
			expectedO2N:        []int{0, 1, 2, 3, 4, 5, -1, -1, 6, 7, 8, 9},
		},
		{
			name:               "Only newlines",
			input:              "\n\n\n",
			expectedNormalized: "",
			expectedN2O:        []int{},
			expectedO2N:        []int{-1, -1, -1},
		},
		{
			name:               "Invalid UTF-8 sequence",
			input:              "hello\xed\xbe\xadworld", // Invalid UTF-8 for U+FDD0
			expectedNormalized: "hello" + string(utf8.RuneError) + "world",
			expectedN2O:        []int{0, 1, 2, 3, 4, 5, 5, 5, 5, 8, 9, 10, 11}, // Mapping for '' points to start of invalid sequence
			expectedO2N:        []int{0, 1, 2, 3, 4, 5, -1, -1, -1, 6, 7, 8, 9},
		},
		{
			name:               "Complex mapping check",
			input:              "  a b  c   d  ",
			expectedNormalized: "a b c d",
			expectedN2O:        []int{2, 3, 4, 6, 7, 8, 11, 12, 13},
			expectedO2N:        []int{-1, -1, 0, 1, -1, 2, 3, -1, -1, 4, 5, -1, -1, -1, 6, 7, 8, -1, -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, n2o, o2n := NormalizeWhitespace(tt.input)

			if normalized != tt.expectedNormalized {
				t.Errorf("Normalized mismatch: got %q, want %q", normalized, tt.expectedNormalized)
			}

			// For n2o and o2n, we'll do a more robust check if they are not empty
			if len(tt.expectedN2O) > 0 && !reflect.DeepEqual(n2o, tt.expectedN2O) {
				t.Errorf("n2o mapping mismatch:\n  got: %v\n  want: %v", n2o, tt.expectedN2O)
			}
			if len(tt.expectedO2N) > 0 && !reflect.DeepEqual(o2n, tt.expectedO2N) {
				t.Errorf("o2n mapping mismatch:\n  got: %v\n  want: %v", o2n, tt.expectedO2N)
			}
		})
	}
}
