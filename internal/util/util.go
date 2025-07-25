package util

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/garaekz/fileman/internal/model"
)

// --- String and Slice Helpers ---

// Splice replaces a slice of bytes with another slice.
func Splice(b []byte, start, end int, replacement []byte) []byte {
	var buf bytes.Buffer
	buf.Grow(len(b) - (end - start) + len(replacement))
	buf.Write(b[:start])
	buf.Write(replacement)
	buf.Write(b[end:])
	return buf.Bytes()
}

// ReverseChanges reverses a slice of Change structs in place.
func ReverseChanges(cs []model.Change) {
	for i, j := 0, len(cs)-1; i < j; i, j = i+1, j-1 {
		cs[i], cs[j] = cs[j], cs[i]
	}
}

// TakeIndent extracts the leading whitespace from a string.
func TakeIndent(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == ' ' || r == '\t' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return b.String()
}

// SumChangedBytes calculates the absolute difference in bytes from a list of changes.
func SumChangedBytes(ch []model.Change) int {
	total := 0
	for _, c := range ch {
		d := len(c.New) - len(c.Original)
		if d < 0 {
			d = -d
		}
		total += d
	}
	return total
}

// NormalizeWhitespace colapsa cualquier secuencia de espacios Unicode en un único ' ',
// elimina espacios líderes y finales, y devuelve los mapas bidireccionales de índices
// *en bytes*:
//
//  1. normalizedToOriginal[nIdx] = índice byte en s que "originó" el byte nIdx en la normalizada
//     (para runas multibyte, cada byte normalizado apunta al byte inicial del rune original).
//     Para el ' ' colapsado, apunta al primer byte de la secuencia original de whitespace.
//  2. originalToNormalized[oIdx] = índice byte en la normalizada que corresponde a oIdx;
//     si el byte original fue colapsado/recortado, será -1.
//
// Nota: No hacemos TrimSpace al final; evitamos emitir espacios líderes/trailing desde el inicio,
// así no tenemos que reindexar nada después.
func NormalizeWhitespace(
	s string,
) (normalized string, normalizedToOriginal []int, originalToNormalized []int) {
	if len(s) == 0 {
		return "", nil, nil
	}

	var result []byte
	originalToNormalized = make([]int, len(s))
	for i := range originalToNormalized {
		originalToNormalized[i] = -1
	}

	i := 0
	normalizedPos := 0
	hasContent := false

	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])

		if r == utf8.RuneError {
			// Handle consecutive invalid UTF-8 bytes as one replacement character
			invalidStart := i
			for i < len(s) {
				r2, size2 := utf8.DecodeRuneInString(s[i:])
				if r2 != utf8.RuneError {
					break
				}
				i += size2
			}

			// Emit one replacement character for the entire invalid sequence
			replacement := []byte(string(utf8.RuneError))
			result = append(result, replacement...)

			// Map the replacement bytes to the start of the invalid sequence
			for range replacement {
				normalizedToOriginal = append(normalizedToOriginal, invalidStart)
			}

			// Map first original invalid byte to start of replacement
			originalToNormalized[invalidStart] = normalizedPos
			// Other invalid bytes map to -1 (they were combined)
			for j := invalidStart + 1; j < i; j++ {
				originalToNormalized[j] = -1
			}

			normalizedPos += len(replacement)
			hasContent = true
			continue
		}

		if unicode.IsSpace(r) {
			// Skip leading whitespace
			if !hasContent {
				i += size
				continue
			}

			// Find end of whitespace sequence
			wsEnd := i
			for wsEnd < len(s) {
				r2, size2 := utf8.DecodeRuneInString(s[wsEnd:])
				if r2 == utf8.RuneError || !unicode.IsSpace(r2) {
					break
				}
				wsEnd += size2
			}

			// If we're at the end, this is trailing whitespace - skip it
			if wsEnd >= len(s) {
				break
			}

			// Emit single space for internal whitespace
			result = append(result, ' ')

			// Map the single space to any byte in the whitespace sequence
			// Let's use the first byte for consistency
			normalizedToOriginal = append(normalizedToOriginal, i)
			originalToNormalized[i] = normalizedPos

			normalizedPos++
			i = wsEnd
			continue
		}

		// Non-whitespace character
		runeBytes := make([]byte, size)
		utf8.EncodeRune(runeBytes, r)
		result = append(result, runeBytes...)

		// Map each byte
		for j := range size {
			normalizedToOriginal = append(normalizedToOriginal, i+j)
			originalToNormalized[i+j] = normalizedPos + j
		}

		normalizedPos += size
		hasContent = true
		i += size
	}

	return string(result), normalizedToOriginal, originalToNormalized
}
