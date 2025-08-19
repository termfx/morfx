package util

import (
	"testing"

	"github.com/termfx/morfx/internal/model"
)

func TestSplice(t *testing.T) {
	tests := []struct {
		name        string
		b           []byte
		start       int
		end         int
		replacement []byte
		expected    []byte
	}{
		{
			name:        "Replace in middle",
			b:           []byte("abcdefg"),
			start:       2,
			end:         5,
			replacement: []byte("XYZ"),
			expected:    []byte("abXYZfg"),
		},
		{
			name:        "Insert at beginning",
			b:           []byte("def"),
			start:       0,
			end:         0,
			replacement: []byte("abc"),
			expected:    []byte("abcdef"),
		},
		{
			name:        "Insert at end",
			b:           []byte("abc"),
			start:       3,
			end:         3,
			replacement: []byte("def"),
			expected:    []byte("abcdef"),
		},
		{
			name:        "Delete in middle",
			b:           []byte("abcdefg"),
			start:       2,
			end:         5,
			replacement: []byte(""),
			expected:    []byte("abfg"),
		},
		{
			name:        "Replace entire slice",
			b:           []byte("abcdefg"),
			start:       0,
			end:         7,
			replacement: []byte("XYZ"),
			expected:    []byte("XYZ"),
		},
		{
			name:        "Empty original, insert",
			b:           []byte(""),
			start:       0,
			end:         0,
			replacement: []byte("abc"),
			expected:    []byte("abc"),
		},
		{
			name:        "Empty replacement, empty original",
			b:           []byte(""),
			start:       0,
			end:         0,
			replacement: []byte(""),
			expected:    []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Splice(tt.b, tt.start, tt.end, tt.replacement)
			if string(result) != string(tt.expected) {
				t.Errorf(
					"Splice(%q, %d, %d, %q) = %q; want %q",
					tt.b,
					tt.start,
					tt.end,
					tt.replacement,
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestReverseChanges(t *testing.T) {
	tests := []struct {
		name     string
		input    []model.Change
		expected []model.Change
	}{
		{
			name:     "Empty slice",
			input:    []model.Change{},
			expected: []model.Change{},
		},
		{
			name: "Single element",
			input: []model.Change{
				{RuleID: "1"},
			},
			expected: []model.Change{
				{RuleID: "1"},
			},
		},
		{
			name: "Multiple elements",
			input: []model.Change{
				{RuleID: "1"},
				{RuleID: "2"},
				{RuleID: "3"},
			},
			expected: []model.Change{
				{RuleID: "3"},
				{RuleID: "2"},
				{RuleID: "1"},
			},
		},
		{
			name: "Even number of elements",
			input: []model.Change{
				{RuleID: "A"},
				{RuleID: "B"},
				{RuleID: "C"},
				{RuleID: "D"},
			},
			expected: []model.Change{
				{RuleID: "D"},
				{RuleID: "C"},
				{RuleID: "B"},
				{RuleID: "A"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to avoid modifying the original input slice
			inputCopy := make([]model.Change, len(tt.input))
			copy(inputCopy, tt.input)

			ReverseChanges(inputCopy)
			if len(inputCopy) != len(tt.expected) {
				t.Fatalf("Length mismatch: got %d, want %d", len(inputCopy), len(tt.expected))
			}
			for i := range inputCopy {
				if inputCopy[i].RuleID != tt.expected[i].RuleID { // Compare a field to keep it simple
					t.Errorf("ReverseChanges() = %v; want %v", inputCopy, tt.expected)
					break
				}
			}
		})
	}
}

func TestTakeIndent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No indent",
			input:    "hello",
			expected: "",
		},
		{
			name:     "Space indent",
			input:    "  hello",
			expected: "  ",
		},
		{
			name:     "Tab indent",
			input:    "\t\thello",
			expected: "\t\t",
		},
		{
			name:     "Mixed indent",
			input:    " \t hello",
			expected: " \t ",
		},
		{
			name:     "Only indent",
			input:    "    ",
			expected: "    ",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Newline in indent (should stop at newline)",
			input:    "  \nhello",
			expected: "  ",
		},
		{
			name:     "Non-whitespace immediately",
			input:    "abc",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TakeIndent(tt.input)
			if result != tt.expected {
				t.Errorf("TakeIndent(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSumChangedBytes(t *testing.T) {
	tests := []struct {
		name     string
		changes  []model.Change
		expected int
	}{
		{
			name:     "Empty changes",
			changes:  []model.Change{},
			expected: 0,
		},
		{
			name: "Single replacement (increase)",
			changes: []model.Change{
				{Original: "a", New: "abc"},
			},
			expected: 2,
		},
		{
			name: "Single replacement (decrease)",
			changes: []model.Change{
				{Original: "abc", New: "a"},
			},
			expected: 2,
		},
		{
			name: "Single replacement (no change)",
			changes: []model.Change{
				{Original: "abc", New: "abc"},
			},
			expected: 0,
		},
		{
			name: "Multiple replacements",
			changes: []model.Change{
				{Original: "a", New: "aa"},  // +1
				{Original: "bb", New: "b"},  // -1
				{Original: "c", New: "ccc"}, // +2
			},
			expected: 4, // |1| + |-1| + |2| = 1 + 1 + 2 = 4
		},
		{
			name: "Mixed changes with zero",
			changes: []model.Change{
				{Original: "x", New: "y"},     // +0
				{Original: "foo", New: ""},    // -3
				{Original: "bar", New: "baz"}, // +0
			},
			expected: 3, // |0| + |-3| + |0| = 0 + 3 + 0 = 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SumChangedBytes(tt.changes)
			if result != tt.expected {
				t.Errorf("SumChangedBytes(%v) = %d; want %d", tt.changes, result, tt.expected)
			}
		})
	}
}
