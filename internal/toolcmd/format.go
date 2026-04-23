package toolcmd

import "strings"

// FormatConfidence renders a 10-cell bar for a normalized confidence score.
func FormatConfidence(score float64) string {
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	filled := int(score * 10)
	if filled > 10 {
		filled = 10
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}
