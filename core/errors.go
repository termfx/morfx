package core

import "errors"

// ErrNoMatchesFound indicates that a transform query did not match anything in
// the current source. Batch file operations treat this as a no-op, not a
// failure, because most files in a scope will legitimately have zero matches.
var ErrNoMatchesFound = errors.New("no matches found for target")
