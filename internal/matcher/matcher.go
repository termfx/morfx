package matcher

// Result represents a byte span (inclusive start, exclusive end) in the source.
type Result struct {
	Start int
	End   int
}

// Matcher abstracts any engine (regex, AST, etc.) that can return byte spans
// of matches in the provided source.
type Matcher interface {
	Find(src []byte) ([]Result, error)
}
