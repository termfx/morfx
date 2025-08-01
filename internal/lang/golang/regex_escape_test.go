package golang

import (
	"testing"
)

// TestReEscape tests regex character escaping functionality
func TestReEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no_special_chars",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "dot_character",
			input:    "hello.world",
			expected: "hello\\.world",
		},
		{
			name:     "plus_character",
			input:    "test+123",
			expected: "test\\+123",
		},
		{
			name:     "asterisk_and_dot",
			input:    "regex.*",
			expected: "regex\\.\\*",
		},
		{
			name:     "forward_slash",
			input:    "path/to/file",
			expected: "path/to/file", // Forward slash is not escaped by regexp.QuoteMeta
		},
		{
			name:     "square_brackets",
			input:    "[brackets]",
			expected: "\\[brackets\\]",
		},
		{
			name:     "parentheses",
			input:    "(parentheses)",
			expected: "\\(parentheses\\)",
		},
		{
			name:     "curly_braces",
			input:    "{braces}",
			expected: "\\{braces\\}",
		},
		{
			name:     "caret_and_dollar",
			input:    "^start$",
			expected: "\\^start\\$",
		},
		{
			name:     "backslash",
			input:    "\\backslash",
			expected: "\\\\backslash",
		},
		{
			name:     "pipe_character",
			input:    "pipe|char",
			expected: "pipe\\|char",
		},
		{
			name:     "question_mark",
			input:    "question?mark",
			expected: "question\\?mark",
		},
		{
			name:     "empty_string",
			input:    "",
			expected: "",
		},
		{
			name:     "all_special_chars",
			input:    ".+*?[](){}|^$\\",
			expected: "\\.\\+\\*\\?\\[\\]\\(\\)\\{\\}\\|\\^\\$\\\\",
		},
		{
			name:     "go_package_path",
			input:    "github.com/pkg/errors",
			expected: "github\\.com/pkg/errors",
		},
		{
			name:     "function_name_with_dots",
			input:    "fmt.Printf",
			expected: "fmt\\.Printf",
		},
		{
			name:     "regex_pattern",
			input:    "[a-zA-Z0-9]+",
			expected: "\\[a-zA-Z0-9\\]\\+",
		},
		{
			name:     "complex_pattern",
			input:    "^(test|example).*\\d+$",
			expected: "\\^\\(test\\|example\\)\\.\\*\\\\d\\+\\$",
		},
		{
			name:     "unicode_characters",
			input:    "测试.函数",
			expected: "测试\\.函数",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reEscape(tt.input)
			if result != tt.expected {
				t.Errorf("reEscape(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestReEscape_EdgeCases tests edge cases and boundary conditions
func TestReEscape_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "single_dot",
			input:       ".",
			description: "Single dot should be escaped",
		},
		{
			name:        "single_asterisk",
			input:       "*",
			description: "Single asterisk should be escaped",
		},
		{
			name:        "multiple_dots",
			input:       "...",
			description: "Multiple dots should all be escaped",
		},
		{
			name:        "mixed_special_chars",
			input:       "a.b+c*d?e",
			description: "Mixed special characters should be escaped",
		},
		{
			name:        "only_special_chars",
			input:       ".*+?",
			description: "String with only special characters",
		},
		{
			name:        "whitespace_with_special",
			input:       "test .* pattern",
			description: "Whitespace should be preserved, special chars escaped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reEscape(tt.input)

			// Verify that the result is different from input if input contains special chars
			hasSpecialChars := false
			specialChars := ".+*?[](){}|^$\\"
			for _, char := range tt.input {
				for _, special := range specialChars {
					if char == special {
						hasSpecialChars = true
						break
					}
				}
				if hasSpecialChars {
					break
				}
			}

			if hasSpecialChars && result == tt.input {
				t.Errorf("reEscape(%q) should escape special characters but returned unchanged result", tt.input)
			}

			t.Logf("%s: %q -> %q", tt.description, tt.input, result)
		})
	}
}

// TestReEscape_RealWorldScenarios tests real-world usage scenarios
func TestReEscape_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		scenario string
	}{
		{
			name:     "go_import_path",
			input:    "github.com/gorilla/mux",
			expected: "github\\.com/gorilla/mux",
			scenario: "Go import path with dots",
		},
		{
			name:     "method_call",
			input:    "http.ListenAndServe",
			expected: "http\\.ListenAndServe",
			scenario: "Method call with package prefix",
		},
		{
			name:     "file_extension",
			input:    "main.go",
			expected: "main\\.go",
			scenario: "File name with extension",
		},
		{
			name:     "version_string",
			input:    "v1.2.3",
			expected: "v1\\.2\\.3",
			scenario: "Semantic version string",
		},
		{
			name:     "url_pattern",
			input:    "https://api.example.com/v1",
			expected: "https://api\\.example\\.com/v1",
			scenario: "URL with dots",
		},
		{
			name:     "regex_class",
			input:    "[a-zA-Z0-9_]",
			expected: "\\[a-zA-Z0-9_\\]",
			scenario: "Character class pattern",
		},
		{
			name:     "function_signature",
			input:    "func(string) error",
			expected: "func\\(string\\) error",
			scenario: "Function signature with parentheses",
		},
		{
			name:     "generic_type",
			input:    "map[string]interface{}",
			expected: "map\\[string\\]interface\\{\\}",
			scenario: "Generic type with brackets and braces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reEscape(tt.input)
			if result != tt.expected {
				t.Errorf("reEscape(%q) = %q, want %q (scenario: %s)", tt.input, result, tt.expected, tt.scenario)
			}
			t.Logf("%s: %q -> %q", tt.scenario, tt.input, result)
		})
	}
}

// BenchmarkReEscape benchmarks the regex escaping performance
func BenchmarkReEscape(b *testing.B) {
	benchmarks := []struct {
		name  string
		input string
	}{
		{"no_special_chars", "helloworld"},
		{"few_special_chars", "hello.world"},
		{"many_special_chars", ".*+?[](){}|^$\\"},
		{"long_string_no_special", string(make([]byte, 1000))},
		{"long_string_with_dots", "github.com/very/long/package/path/with/many/dots"},
		{"complex_pattern", "^(test|example).*\\d+$"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for b.Loop() {
				reEscape(bm.input)
			}
		})
	}
}
