package matcher

import "regexp"

// RegexMatcher implements Matcher using a compiled *regexp.Regexp.
type RegexMatcher struct {
	re *regexp.Regexp
}

// NewRegex returns a RegexMatcher with the given pattern already compiled.
// Caller is responsible for adding flags like (?m) or (?s) beforehand.
func NewRegex(pattern string) (*RegexMatcher, error) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RegexMatcher{re: r}, nil
}

// Find returns all match spans for the compiled expression.
func (r *RegexMatcher) Find(src []byte) ([]Result, error) {
	idx := r.re.FindAllIndex(src, -1)
	out := make([]Result, len(idx))
	for i, span := range idx {
		out[i] = Result{Start: span[0], End: span[1]}
	}
	return out, nil
}
