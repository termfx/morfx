package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFuzzyResolver_NewFuzzyResolver tests the creation of a new fuzzy resolver
func TestFuzzyResolver_NewFuzzyResolver(t *testing.T) {
	fr := NewFuzzyResolver()
	require.NotNil(t, fr)
	assert.Equal(t, 3, fr.maxDistance)
	assert.Len(t, fr.heuristics, 8) // Should have 8 default heuristics
}

// TestFuzzyResolver_ExactMatch tests exact string matching heuristic
func TestFuzzyResolver_ExactMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expScore    float64
		expDistance int
	}{
		{
			name:        "identical strings",
			original:    "hello",
			candidate:   "hello",
			expScore:    1.0,
			expDistance: 0,
		},
		{
			name:        "different strings",
			original:    "hello",
			candidate:   "world",
			expScore:    0.0,
			expDistance: 10, // len("hello") + len("world")
		},
		{
			name:        "empty strings",
			original:    "",
			candidate:   "",
			expScore:    1.0,
			expDistance: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := exactMatch(tt.original, tt.candidate)
			assert.Equal(t, tt.expScore, score)
			assert.Equal(t, tt.expDistance, distance)
		})
	}
}

// TestFuzzyResolver_CaseInsensitiveMatch tests case-insensitive matching heuristic
func TestFuzzyResolver_CaseInsensitiveMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expScore    float64
		expDistance int
	}{
		{
			name:        "same case",
			original:    "hello",
			candidate:   "hello",
			expScore:    1.0,
			expDistance: 0,
		},
		{
			name:        "different case",
			original:    "hello",
			candidate:   "HELLO",
			expScore:    1.0,
			expDistance: 5, // All 5 characters differ in case
		},
		{
			name:        "mixed case",
			original:    "Hello",
			candidate:   "hELLO",
			expScore:    1.0,
			expDistance: 5, // All characters differ in case
		},
		{
			name:        "completely different",
			original:    "hello",
			candidate:   "world",
			expScore:    0.0,
			expDistance: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := caseInsensitiveMatch(tt.original, tt.candidate)
			assert.Equal(t, tt.expScore, score)
			assert.Equal(t, tt.expDistance, distance)
		})
	}
}

// TestFuzzyResolver_LevenshteinMatch tests Levenshtein distance-based matching
func TestFuzzyResolver_LevenshteinMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expDistance int
		minScore    float64 // Minimum expected score
	}{
		{
			name:        "identical strings",
			original:    "hello",
			candidate:   "hello",
			expDistance: 0,
			minScore:    1.0,
		},
		{
			name:        "single character difference",
			original:    "hello",
			candidate:   "hallo",
			expDistance: 1,
			minScore:    0.8, // 1 - 1/5 = 0.8
		},
		{
			name:        "insertion",
			original:    "hello",
			candidate:   "helllo",
			expDistance: 1,
			minScore:    0.83, // 1 - 1/6 ≈ 0.83
		},
		{
			name:        "deletion",
			original:    "hello",
			candidate:   "hllo",
			expDistance: 1,
			minScore:    0.8, // 1 - 1/5 = 0.8
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := levenshteinMatch(tt.original, tt.candidate)
			assert.Equal(t, tt.expDistance, distance)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, 1.0)
		})
	}
}

// TestFuzzyResolver_SubstringMatch tests substring containment matching
func TestFuzzyResolver_SubstringMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expScore    float64
		expDistance int
	}{
		{
			name:        "original contained in candidate",
			original:    "hello",
			candidate:   "say hello world",
			expScore:    float64(5) / float64(15), // len("hello") / len("say hello world")
			expDistance: 10,                       // 15 - 5
		},
		{
			name:        "candidate contained in original",
			original:    "say hello world",
			candidate:   "hello",
			expScore:    float64(5) / float64(15), // len("hello") / len("say hello world")
			expDistance: 10,                       // 15 - 5
		},
		{
			name:        "no containment",
			original:    "hello",
			candidate:   "world",
			expScore:    0.0,
			expDistance: 10, // 5 + 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := substringMatch(tt.original, tt.candidate)
			assert.InDelta(t, tt.expScore, score, 0.01)
			assert.Equal(t, tt.expDistance, distance)
		})
	}
}

// TestFuzzyResolver_PrefixMatch tests prefix matching
func TestFuzzyResolver_PrefixMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expScore    float64
		expDistance int
	}{
		{
			name:        "common prefix",
			original:    "hello",
			candidate:   "help",
			expScore:    float64(3) / float64(5), // "hel" is common, max len is 5
			expDistance: 2,                       // 5 - 3
		},
		{
			name:        "no common prefix",
			original:    "hello",
			candidate:   "world",
			expScore:    0.0,
			expDistance: 10, // 5 + 5
		},
		{
			name:        "identical strings",
			original:    "hello",
			candidate:   "hello",
			expScore:    1.0,
			expDistance: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := prefixMatch(tt.original, tt.candidate)
			assert.InDelta(t, tt.expScore, score, 0.01)
			assert.Equal(t, tt.expDistance, distance)
		})
	}
}

// TestFuzzyResolver_CamelCaseMatch tests CamelCase abbreviation matching
func TestFuzzyResolver_CamelCaseMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expScore    float64
		expDistance int
	}{
		{
			name:        "camelCase abbreviation",
			original:    "GHW",
			candidate:   "GetHelloWorld",
			expScore:    float64(3) / float64(13), // len("GHW") / len("GetHelloWorld")
			expDistance: 10,                       // 13 - 3
		},
		{
			name:        "case insensitive match",
			original:    "ghw",
			candidate:   "GetHelloWorld",
			expScore:    float64(3) / float64(13),
			expDistance: 10,
		},
		{
			name:        "no match",
			original:    "XYZ",
			candidate:   "GetHelloWorld",
			expScore:    0.0,
			expDistance: 16, // 3 + 13
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := camelCaseMatch(tt.original, tt.candidate)
			assert.InDelta(t, tt.expScore, score, 0.01)
			assert.Equal(t, tt.expDistance, distance)
		})
	}
}

// TestFuzzyResolver_AcronymMatch tests acronym matching
func TestFuzzyResolver_AcronymMatch(t *testing.T) {
	tests := []struct {
		name        string
		original    string
		candidate   string
		expScore    float64
		expDistance int
	}{
		{
			name:        "acronym match",
			original:    "GHW",
			candidate:   "Get Hello World",
			expScore:    float64(3) / float64(15), // len("GHW") / len("Get Hello World")
			expDistance: 12,                       // 15 - 3
		},
		{
			name:        "case insensitive",
			original:    "ghw",
			candidate:   "Get Hello World",
			expScore:    float64(3) / float64(15),
			expDistance: 12,
		},
		{
			name:        "no match",
			original:    "XYZ",
			candidate:   "Get Hello World",
			expScore:    0.0,
			expDistance: 18, // 3 + 15
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, distance := acronymMatch(tt.original, tt.candidate)
			assert.InDelta(t, tt.expScore, score, 0.01)
			assert.Equal(t, tt.expDistance, distance)
		})
	}
}

// TestFuzzyResolver_GenerateQueryVariations tests query variation generation
func TestFuzzyResolver_GenerateQueryVariations(t *testing.T) {
	fr := NewFuzzyResolver()

	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:  "simple query",
			query: "hello",
			expected: []string{
				"hello",
				"hello", // lowercase (duplicate, should be removed)
				"HELLO",
				"Hello",
				"*hello*",
				"hello*",
				"*hello",
				"H", // CamelCase abbreviation
				"h", // lowercase abbreviation
			},
		},
		{
			name:  "query with prefix",
			query: "getName",
			expected: []string{
				"getName",
				"getname",
				"GETNAME",
				"Getname",
				"getName",  // camelCase (same as original)
				"GetName",  // PascalCase
				"get_name", // snake_case
				"Name",     // remove "get" prefix
				"name",     // lowercase remaining
				"*getName*",
				"getName*",
				"*getName",
				"GN", // CamelCase abbreviation
				"gn", // lowercase abbreviation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variations := fr.generateQueryVariations(tt.query)

			// Check that all expected variations are present
			for _, expected := range tt.expected {
				if expected != tt.query { // Skip checking duplicates of original
					assert.Contains(t, variations, expected, "Expected variation %s not found", expected)
				}
			}

			// Check that original query is first
			assert.Equal(t, tt.query, variations[0], "Original query should be first")

			// Check no duplicates
			seen := make(map[string]bool)
			for _, variation := range variations {
				assert.False(t, seen[variation], "Duplicate variation found: %s", variation)
				seen[variation] = true
			}
		})
	}
}

// TestFuzzyResolver_CaseConversions tests case conversion helper functions
func TestFuzzyResolver_CaseConversions(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		camelCase  string
		pascalCase string
		snakeCase  string
	}{
		{
			name:       "simple word",
			input:      "hello",
			camelCase:  "hello",
			pascalCase: "Hello",
			snakeCase:  "hello",
		},
		{
			name:       "multiple words",
			input:      "hello world",
			camelCase:  "helloWorld",
			pascalCase: "HelloWorld",
			snakeCase:  "hello world", // No change for space-separated
		},
		{
			name:       "camelCase input",
			input:      "helloWorld",
			camelCase:  "helloworld", // Converts to lowercase then processes
			pascalCase: "Helloworld",
			snakeCase:  "hello_world",
		},
		{
			name:       "PascalCase input",
			input:      "HelloWorld",
			camelCase:  "helloworld",
			pascalCase: "Helloworld",
			snakeCase:  "_hello_world", // Leading underscore due to first uppercase
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.camelCase, toCamelCase(tt.input))
			assert.Equal(t, tt.pascalCase, toPascalCase(tt.input))
			assert.Equal(t, tt.snakeCase, toSnakeCase(tt.input))
		})
	}
}

// TestFuzzyResolver_ExtractAbbreviations tests abbreviation extraction functions
func TestFuzzyResolver_ExtractAbbreviations(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		camelCaseAbbrev string
		acronym         string
	}{
		{
			name:            "camelCase",
			input:           "getHelloWorldString",
			camelCaseAbbrev: "GHWS",              // First letter + uppercase letters
			acronym:         "GHELLOWORLDSTRING", // First letter of each "word" (no separators)
		},
		{
			name:            "PascalCase",
			input:           "GetHelloWorldString",
			camelCaseAbbrev: "GHWS",
			acronym:         "GHELLOWORLDSTRING",
		},
		{
			name:            "space separated",
			input:           "Get Hello World",
			camelCaseAbbrev: "GHW", // First letter + uppercase letters
			acronym:         "GHW", // First letter of each word
		},
		{
			name:            "underscore separated",
			input:           "get_hello_world",
			camelCaseAbbrev: "G",   // Only first letter (no uppercase)
			acronym:         "GHW", // First letter of each word
		},
		{
			name:            "single word",
			input:           "hello",
			camelCaseAbbrev: "H",
			acronym:         "H",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.camelCaseAbbrev, extractCamelCaseAbbreviation(tt.input))
			assert.Equal(t, tt.acronym, extractAcronym(tt.input))
		})
	}
}

// TestFuzzyResolver_CalculateConfidence tests confidence calculation
func TestFuzzyResolver_CalculateConfidence(t *testing.T) {
	fr := NewFuzzyResolver()

	tests := []struct {
		name        string
		score       float64
		distance    int
		maxDistance int
		expected    float64
	}{
		{
			name:        "perfect match",
			score:       1.0,
			distance:    0,
			maxDistance: 3,
			expected:    1.0, // 1.0 * (1.0 - 0/4)
		},
		{
			name:        "good match with small distance",
			score:       0.8,
			distance:    1,
			maxDistance: 3,
			expected:    0.6, // 0.8 * (1.0 - 1/4) = 0.8 * 0.75 = 0.6
		},
		{
			name:        "poor match with max distance",
			score:       0.5,
			distance:    3,
			maxDistance: 3,
			expected:    0.125, // 0.5 * (1.0 - 3/4) = 0.5 * 0.25 = 0.125
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := fr.calculateConfidence(tt.score, tt.distance, tt.maxDistance)
			assert.InDelta(t, tt.expected, confidence, 0.001)
		})
	}
}

// TestFuzzyResolver_LevenshteinDistance tests the Levenshtein distance calculation
func TestFuzzyResolver_LevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{
			name:     "identical strings",
			s1:       "hello",
			s2:       "hello",
			expected: 0,
		},
		{
			name:     "single substitution",
			s1:       "hello",
			s2:       "hallo",
			expected: 1,
		},
		{
			name:     "single insertion",
			s1:       "hello",
			s2:       "helllo",
			expected: 1,
		},
		{
			name:     "single deletion",
			s1:       "hello",
			s2:       "hllo",
			expected: 1,
		},
		{
			name:     "empty to non-empty",
			s1:       "",
			s2:       "hello",
			expected: 5,
		},
		{
			name:     "non-empty to empty",
			s1:       "hello",
			s2:       "",
			expected: 5,
		},
		{
			name:     "completely different",
			s1:       "hello",
			s2:       "world",
			expected: 4, // Need 4 operations to transform
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := levenshteinDistance(tt.s1, tt.s2)
			assert.Equal(t, tt.expected, distance)
		})
	}
}

// TestFuzzyResolver_UtilityFunctions tests utility functions
func TestFuzzyResolver_UtilityFunctions(t *testing.T) {
	// Test min function
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(2, 1))
	assert.Equal(t, 1, min(1, 2, 3))
	assert.Equal(t, 1, min(3, 2, 1))

	// Test maxInt function
	assert.Equal(t, 2, maxInt(1, 2))
	assert.Equal(t, 2, maxInt(2, 1))

	// Test maxFloat64 function
	assert.Equal(t, 2.5, maxFloat64(1.5, 2.5))
	assert.Equal(t, 2.5, maxFloat64(2.5, 1.5))

	// Test removeDuplicates function
	input := []string{"a", "b", "a", "c", "b", "d"}
	expected := []string{"a", "b", "c", "d"}
	result := removeDuplicates(input)
	assert.Equal(t, expected, result)

	// Test longestCommonPrefix
	assert.Equal(t, 3, longestCommonPrefix("hello", "help"))
	assert.Equal(t, 0, longestCommonPrefix("hello", "world"))
	assert.Equal(t, 5, longestCommonPrefix("hello", "hello"))

	// Test longestCommonSuffix
	assert.Equal(t, 3, longestCommonSuffix("testing", "running"))
	assert.Equal(t, 0, longestCommonSuffix("hello", "world"))
	assert.Equal(t, 5, longestCommonSuffix("hello", "hello"))
}
