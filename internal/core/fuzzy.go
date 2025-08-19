package core

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
)

// FuzzyResolver implements deterministic fuzzy matching with heuristics
type FuzzyResolver struct {
	maxDistance int
	heuristics  []Heuristic
}

// Heuristic represents a fuzzy matching strategy
type Heuristic struct {
	Name        string
	Description string
	Weight      float64
	Apply       func(original, candidate string) (score float64, distance int)
}

// NewFuzzyResolver creates a new fuzzy resolver with default heuristics
func NewFuzzyResolver() *FuzzyResolver {
	return &FuzzyResolver{
		maxDistance: 3, // Default max edit distance
		heuristics:  getDefaultHeuristics(),
	}
}

// Resolve attempts to find fuzzy matches when exact matching fails
func (fr *FuzzyResolver) Resolve(
	tree *sitter.Tree,
	query string,
	source []byte,
	provider FuzzyProvider,
	maxDistance int,
) ([]*sitter.Node, *FuzzyMatch, error) {
	if maxDistance > 0 {
		fr.maxDistance = maxDistance
	}

	// Generate query variations using heuristics
	variations := fr.generateQueryVariations(query)

	// Score each variation
	type candidateScore struct {
		variation  string
		score      float64
		distance   int
		heuristics []string
		confidence float64
		nodes      []*sitter.Node
	}

	var scores []candidateScore

	// Test each variation and score it
	for _, variation := range variations {
		// Try to execute the variation as a query
		nodes, err := fr.executeVariation(tree, variation, provider)
		if err != nil || len(nodes) == 0 {
			continue // Skip variations that don't match
		}

		// Score this variation using all heuristics
		totalScore := 0.0
		totalWeight := 0.0
		minDistance := fr.maxDistance + 1
		appliedHeuristics := []string{}

		for _, heuristic := range fr.heuristics {
			score, distance := heuristic.Apply(query, variation)
			if distance <= fr.maxDistance {
				totalScore += score * heuristic.Weight
				totalWeight += heuristic.Weight
				if distance < minDistance {
					minDistance = distance
				}
				appliedHeuristics = append(appliedHeuristics, heuristic.Name)
			}
		}

		if totalWeight > 0 && minDistance <= fr.maxDistance {
			finalScore := totalScore / totalWeight
			confidence := fr.calculateConfidence(finalScore, minDistance, fr.maxDistance)

			scores = append(scores, candidateScore{
				variation:  variation,
				score:      finalScore,
				distance:   minDistance,
				heuristics: appliedHeuristics,
				confidence: confidence,
				nodes:      nodes,
			})
		}
	}

	if len(scores) == 0 {
		return nil, &FuzzyMatch{Used: false}, fmt.Errorf("no fuzzy matches found for query: %s", query)
	}

	// Sort by score (descending), then by distance (ascending) for deterministic results
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score != scores[j].score {
			return scores[i].score > scores[j].score
		}
		if scores[i].distance != scores[j].distance {
			return scores[i].distance < scores[j].distance
		}
		// For deterministic ordering when scores and distances are equal
		return scores[i].variation < scores[j].variation
	})

	best := scores[0]
	fuzzyMatch := &FuzzyMatch{
		Used:          true,
		OriginalQuery: query,
		ResolvedQuery: best.variation,
		Confidence:    best.confidence,
		Score:         best.score,
		Distance:      best.distance,
		Heuristics:    best.heuristics,
	}

	return best.nodes, fuzzyMatch, nil
}

// executeVariation attempts to execute a query variation
func (fr *FuzzyResolver) executeVariation(tree *sitter.Tree, variation string, provider FuzzyProvider) ([]*sitter.Node, error) {
	// Parse the variation as a DSL query
	dslQuery := Query{Pattern: variation} // Simple pattern-based query

	tsQuery, err := provider.TranslateQuery(&dslQuery)
	if err != nil {
		return nil, err
	}

	// Cast the interface{} back to *sitter.Language
	sitterLang, ok := provider.GetSitterLanguage().(*sitter.Language)
	if !ok {
		return nil, fmt.Errorf("invalid sitter language type")
	}

	q, err := sitter.NewQuery([]byte(tsQuery), sitterLang)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	// Execute query and collect matches
	var anchors []*sitter.Node
	qc.Exec(q, tree.RootNode())
	for {
		match, ok := qc.NextMatch()
		if !ok {
			break
		}

		for _, capture := range match.Captures {
			anchors = append(anchors, capture.Node)
		}
	}

	return anchors, nil
}

// generateQueryVariations creates variations of the original query using heuristics
func (fr *FuzzyResolver) generateQueryVariations(query string) []string {
	variations := []string{}

	// Add the original query first
	variations = append(variations, query)

	// Generate case variations
	variations = append(variations, strings.ToLower(query))
	variations = append(variations, strings.ToUpper(query))
	variations = append(variations, strings.Title(query))

	// Generate CamelCase variations
	if camelCase := toCamelCase(query); camelCase != query {
		variations = append(variations, camelCase)
	}
	if pascalCase := toPascalCase(query); pascalCase != query {
		variations = append(variations, pascalCase)
	}
	if snakeCase := toSnakeCase(query); snakeCase != query {
		variations = append(variations, snakeCase)
	}

	// Generate partial matches (remove common prefixes/suffixes)
	if len(query) > 3 {
		// Remove potential prefixes
		prefixes := []string{"get", "set", "is", "has", "can", "should", "will", "do", "make", "create"}
		for _, prefix := range prefixes {
			if strings.HasPrefix(strings.ToLower(query), prefix) {
				remaining := query[len(prefix):]
				if len(remaining) > 0 {
					variations = append(variations, remaining)
					variations = append(variations, strings.ToLower(remaining))
				}
			}
		}

		// Remove potential suffixes
		suffixes := []string{"er", "ed", "ing", "s", "es", "ies", "tion", "sion"}
		for _, suffix := range suffixes {
			if strings.HasSuffix(strings.ToLower(query), suffix) {
				remaining := query[:len(query)-len(suffix)]
				if len(remaining) > 0 {
					variations = append(variations, remaining)
				}
			}
		}
	}

	// Generate wildcard variations
	if !strings.Contains(query, "*") {
		variations = append(variations, "*"+query+"*")
		variations = append(variations, query+"*")
		variations = append(variations, "*"+query)
	}

	// Generate abbreviation variations
	if abbrev := extractCamelCaseAbbreviation(query); abbrev != query && len(abbrev) > 0 {
		variations = append(variations, abbrev)
		variations = append(variations, strings.ToLower(abbrev))
	}

	// Generate acronym variations
	if acronym := extractAcronym(query); acronym != query && len(acronym) > 0 {
		variations = append(variations, acronym)
		variations = append(variations, strings.ToLower(acronym))
	}

	return removeDuplicates(variations)
}

// getDefaultHeuristics returns the standard set of fuzzy matching heuristics
func getDefaultHeuristics() []Heuristic {
	return []Heuristic{
		{
			Name:        "exact_match",
			Description: "Exact string match",
			Weight:      1.0,
			Apply:       exactMatch,
		},
		{
			Name:        "case_insensitive",
			Description: "Case-insensitive match",
			Weight:      0.9,
			Apply:       caseInsensitiveMatch,
		},
		{
			Name:        "levenshtein",
			Description: "Levenshtein distance-based matching",
			Weight:      0.8,
			Apply:       levenshteinMatch,
		},
		{
			Name:        "substring",
			Description: "Substring containment matching",
			Weight:      0.7,
			Apply:       substringMatch,
		},
		{
			Name:        "prefix",
			Description: "Prefix matching",
			Weight:      0.6,
			Apply:       prefixMatch,
		},
		{
			Name:        "suffix",
			Description: "Suffix matching",
			Weight:      0.6,
			Apply:       suffixMatch,
		},
		{
			Name:        "camel_case",
			Description: "CamelCase abbreviation matching",
			Weight:      0.5,
			Apply:       camelCaseMatch,
		},
		{
			Name:        "acronym",
			Description: "Acronym matching",
			Weight:      0.4,
			Apply:       acronymMatch,
		},
	}
}

// calculateConfidence computes confidence based on score, distance, and max distance
func (fr *FuzzyResolver) calculateConfidence(score float64, distance, maxDistance int) float64 {
	distanceRatio := 1.0 - float64(distance)/float64(maxDistance+1)
	return score * distanceRatio
}

// Heuristic implementations

// exactMatch checks for exact string equality
func exactMatch(original, candidate string) (float64, int) {
	if original == candidate {
		return 1.0, 0
	}
	return 0.0, len(original) + len(candidate) // Max possible distance
}

// caseInsensitiveMatch checks for case-insensitive equality
func caseInsensitiveMatch(original, candidate string) (float64, int) {
	origLower := strings.ToLower(original)
	candLower := strings.ToLower(candidate)
	if origLower == candLower {
		return 1.0, countCaseDifferences(original, candidate)
	}
	return 0.0, len(original) + len(candidate)
}

// levenshteinMatch computes Levenshtein distance and score
func levenshteinMatch(original, candidate string) (float64, int) {
	distance := levenshteinDistance(original, candidate)
	maxLen := maxInt(len(original), len(candidate))
	if maxLen == 0 {
		return 1.0, 0
	}
	score := 1.0 - float64(distance)/float64(maxLen)
	return maxFloat64(0.0, score), distance
}

// substringMatch checks if one string contains the other
func substringMatch(original, candidate string) (float64, int) {
	origLower := strings.ToLower(original)
	candLower := strings.ToLower(candidate)

	if strings.Contains(candLower, origLower) {
		score := float64(len(original)) / float64(len(candidate))
		distance := len(candidate) - len(original)
		return score, distance
	}
	if strings.Contains(origLower, candLower) {
		score := float64(len(candidate)) / float64(len(original))
		distance := len(original) - len(candidate)
		return score, distance
	}
	return 0.0, len(original) + len(candidate)
}

// prefixMatch checks for prefix matching
func prefixMatch(original, candidate string) (float64, int) {
	origLower := strings.ToLower(original)
	candLower := strings.ToLower(candidate)

	commonPrefix := longestCommonPrefix(origLower, candLower)
	if commonPrefix == 0 {
		return 0.0, len(original) + len(candidate)
	}

	maxLen := maxInt(len(original), len(candidate))
	score := float64(commonPrefix) / float64(maxLen)
	distance := maxLen - commonPrefix
	return score, distance
}

// suffixMatch checks for suffix matching
func suffixMatch(original, candidate string) (float64, int) {
	origLower := strings.ToLower(original)
	candLower := strings.ToLower(candidate)

	commonSuffix := longestCommonSuffix(origLower, candLower)
	if commonSuffix == 0 {
		return 0.0, len(original) + len(candidate)
	}

	maxLen := maxInt(len(original), len(candidate))
	score := float64(commonSuffix) / float64(maxLen)
	distance := maxLen - commonSuffix
	return score, distance
}

// camelCaseMatch checks if original matches CamelCase abbreviation of candidate
func camelCaseMatch(original, candidate string) (float64, int) {
	abbrev := extractCamelCaseAbbreviation(candidate)
	if strings.EqualFold(original, abbrev) {
		score := float64(len(original)) / float64(len(candidate))
		distance := len(candidate) - len(original)
		return score, distance
	}
	return 0.0, len(original) + len(candidate)
}

// acronymMatch checks if original matches acronym of candidate
func acronymMatch(original, candidate string) (float64, int) {
	acronym := extractAcronym(candidate)
	if strings.EqualFold(original, acronym) {
		score := float64(len(original)) / float64(len(candidate))
		distance := len(candidate) - len(original)
		return score, distance
	}
	return 0.0, len(original) + len(candidate)
}

// Helper functions

// countCaseDifferences counts the number of case differences between two strings
func countCaseDifferences(s1, s2 string) int {
	if len(s1) != len(s2) {
		return maxInt(len(s1), len(s2))
	}
	count := 0
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			count++
		}
	}
	return count
}

// levenshteinDistance computes the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// longestCommonPrefix finds the length of the longest common prefix
func longestCommonPrefix(s1, s2 string) int {
	minLen := min(len(s1), len(s2))
	for i := range minLen {
		if s1[i] != s2[i] {
			return i
		}
	}
	return minLen
}

// longestCommonSuffix finds the length of the longest common suffix
func longestCommonSuffix(s1, s2 string) int {
	len1, len2 := len(s1), len(s2)
	minLen := min(len1, len2)
	for i := range minLen {
		if s1[len1-1-i] != s2[len2-1-i] {
			return i
		}
	}
	return minLen
}

// extractCamelCaseAbbreviation extracts uppercase letters from CamelCase
func extractCamelCaseAbbreviation(s string) string {
	if len(s) == 0 {
		return ""
	}

	var result strings.Builder
	// Add first letter
	result.WriteRune(unicode.ToUpper(rune(s[0])))

	// Add uppercase letters (indicating word boundaries in camelCase)
	for i := 1; i < len(s); i++ {
		if unicode.IsUpper(rune(s[i])) {
			result.WriteRune(rune(s[i]))
		}
	}
	return result.String()
}

// extractAcronym extracts the first letter of each camelCase segment or word
func extractAcronym(s string) string {
	if len(s) == 0 {
		return ""
	}

	var result strings.Builder

	// Check if it's camelCase (no separators)
	if !strings.ContainsAny(s, " _-.") {
		// CamelCase: extract first letter of each segment
		result.WriteRune(unicode.ToUpper(rune(s[0])))
		for i := 1; i < len(s); i++ {
			if unicode.IsUpper(rune(s[i])) {
				// Found start of new word, extract all letters until next uppercase or end
				j := i
				for j < len(s) && (j == i || unicode.IsLower(rune(s[j]))) {
					result.WriteRune(unicode.ToUpper(rune(s[j])))
					j++
				}
				i = j - 1 // Skip processed characters
			}
		}
	} else {
		// Space/underscore separated: extract first letter of each word
		words := strings.FieldsFunc(s, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		for _, word := range words {
			if len(word) > 0 {
				result.WriteRune(unicode.ToUpper(rune(word[0])))
			}
		}
	}
	return result.String()
}

// Case conversion helpers

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) == 0 {
		return s
	}

	var result strings.Builder
	for i, word := range words {
		if i == 0 {
			result.WriteString(strings.ToLower(word))
		} else {
			result.WriteString(strings.Title(strings.ToLower(word)))
		}
	}
	return result.String()
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(words) == 0 {
		return s
	}

	var result strings.Builder
	for _, word := range words {
		result.WriteString(strings.Title(strings.ToLower(word)))
	}
	return result.String()
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	if len(s) == 0 {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			} else {
				// For PascalCase, add underscore before first letter
				result.WriteRune('_')
			}
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(strings []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, str := range strings {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}

// Utility functions

// min returns the minimum of two or three integers
func min(a, b int, c ...int) int {
	result := a
	if b < result {
		result = b
	}
	for _, val := range c {
		if val < result {
			result = val
		}
	}
	return result
}

// maxInt returns the maximum of two integers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// maxFloat64 returns the maximum of two float64 values
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
