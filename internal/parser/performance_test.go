package parser

import (
	"testing"
)

// BenchmarkParseHierarchicalQuery tests performance of hierarchical query parsing
func BenchmarkParseHierarchicalQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "class:MyClass > method:test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkParseComplexLogicalQuery tests performance of complex logical operations
func BenchmarkParseComplexLogicalQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "function:test & variable:name"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkParseQueryWithProvider tests performance of provider integration
func BenchmarkParseQueryWithProvider(b *testing.B) {
	parser := NewUniversalParser()
	provider := NewMockProvider()
	input := "fn:test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQueryWithProvider(input, provider)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkParseNegatedQuery tests performance of negated query parsing
func BenchmarkParseNegatedQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "!function:test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkParseWildcardQuery tests performance of wildcard pattern parsing
func BenchmarkParseWildcardQuery(b *testing.B) {
	parser := NewUniversalParser()
	input := "function:test*pattern*"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkIsWildcard tests performance of wildcard detection
func BenchmarkIsWildcard(b *testing.B) {
	parser := NewUniversalParser()
	patterns := []string{
		"test*",
		"*pattern",
		"test*pattern*end",
		"no_wildcard",
		"escaped\\*asterisk",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pattern := patterns[i%len(patterns)]
		_ = parser.IsWildcard(pattern)
	}
}

// BenchmarkNormalizeQuery tests performance of query normalization
func BenchmarkNormalizeQuery(b *testing.B) {
	parser := NewUniversalParser()
	queryStr := "  function:test   type   "

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.NormalizeQuery(queryStr)
	}
}

// BenchmarkValidateQueryComplex tests performance of complex query validation
func BenchmarkValidateQueryComplex(b *testing.B) {
	parser := NewUniversalParser()
	queries := []string{
		"function:test",
		"class:MyClass > method:test",
		"function:test & variable:name",
		"!function:test",
		"function:test*pattern",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		err := parser.ValidateQuery(query)
		if err != nil {
			b.Errorf("Unexpected error for query %s: %v", query, err)
		}
	}
}

// BenchmarkParseQueryStress tests performance under stress with many operations
func BenchmarkParseQueryStress(b *testing.B) {
	parser := NewUniversalParser()
	query := "function:* & variable:*"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(query)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkParseQueryMemoryAllocation tests memory allocation patterns
func BenchmarkParseQueryMemoryAllocation(b *testing.B) {
	parser := NewUniversalParser()
	input := "class:MyClass > method:test & variable:param"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(input)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkProviderDSLTranslation tests performance of DSL translation with provider
func BenchmarkProviderDSLTranslation(b *testing.B) {
	parser := NewUniversalParser()
	provider := NewMockProvider()
	queries := []string{
		"fn:test",
		"var:name",
		"cls:MyClass",
		"fn:test & var:param",
		"cls:MyClass > fn:method",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		_, err := parser.ParseQueryWithProvider(query, provider)
		if err != nil {
			b.Errorf("Unexpected error for query %s: %v", query, err)
		}
	}
}

// BenchmarkConcurrentParsing tests parser performance under concurrent load
func BenchmarkConcurrentParsing(b *testing.B) {
	parser := NewUniversalParser()
	queries := []string{
		"function:test",
		"class:MyClass > method:test",
		"function:test & variable:name",
		"!function:test",
		"function:test*pattern",
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			query := queries[i%len(queries)]
			_, err := parser.ParseQuery(query)
			if err != nil {
				b.Errorf("Unexpected error for query %s: %v", query, err)
			}
			i++
		}
	})
}

// BenchmarkDeepHierarchicalQuery tests performance with deeply nested hierarchical queries
func BenchmarkDeepHierarchicalQuery(b *testing.B) {
	parser := NewUniversalParser()
	query := "class:MyClass > method:test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(query)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkComplexLogicalCombinations tests performance with complex logical combinations
func BenchmarkComplexLogicalCombinations(b *testing.B) {
	parser := NewUniversalParser()
	query := "function:test & variable:name"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseQuery(query)
		if err != nil {
			b.Errorf("Unexpected error: %v", err)
		}
	}
}

// BenchmarkMixedQueryTypes tests performance with mixed query types
func BenchmarkMixedQueryTypes(b *testing.B) {
	parser := NewUniversalParser()
	queries := []string{
		"function:test",
		"class:MyClass > method:test",
		"function:test & variable:name",
		"!class:Test",
		"class:MyClass > method:test",
		"function:a | variable:b",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		_, err := parser.ParseQuery(query)
		if err != nil {
			b.Errorf("Unexpected error for query %s: %v", query, err)
		}
	}
}