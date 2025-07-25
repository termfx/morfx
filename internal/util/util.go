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
	var b strings.Builder
	b.Grow(len(s)) // Pre-allocate for efficiency, though it might shrink

	// Initialize mapping slices
	normalizedToOriginal = make([]int, 0, len(s))
	originalToNormalized = make([]int, len(s))
	for i := range originalToNormalized {
		originalToNormalized[i] = -1 // Mark as unmapped initially
	}

	emittedAnyNonSpace := false   // Tracks if any non-whitespace character has been emitted
	inWhitespaceSequence := false // Tracks if currently inside a sequence of whitespace
	whitespaceSequenceStart := 0  // Byte index of the start of the current whitespace sequence
	normalizedByteIndex := 0      // Current byte index in the normalized string

	for originalByteIndex := 0; originalByteIndex < len(s); {
		r, size := utf8.DecodeRuneInString(s[originalByteIndex:])

		// Handle RuneError (invalid UTF-8 byte sequence)
		if r == utf8.RuneError && size == 1 {
			// Treat as a non-whitespace character to preserve the byte
			if inWhitespaceSequence {
				inWhitespaceSequence = false
				if emittedAnyNonSpace {
					b.WriteByte(' ')
					normalizedToOriginal = append(normalizedToOriginal, whitespaceSequenceStart)
					normalizedByteIndex++
				}
			}
			// Map the invalid byte directly
			originalToNormalized[originalByteIndex] = normalizedByteIndex
			normalizedToOriginal = append(normalizedToOriginal, originalByteIndex)
			b.WriteByte(s[originalByteIndex])
			normalizedByteIndex++
			emittedAnyNonSpace = true
			originalByteIndex += size
			continue
		}

		if unicode.IsSpace(r) {
			if !inWhitespaceSequence {
				inWhitespaceSequence = true
				whitespaceSequenceStart = originalByteIndex
			}
			// Mark original bytes as unmapped (they will be collapsed or trimmed)
			for j := range size {
				originalToNormalized[originalByteIndex+j] = -1
			}
		} else {
			// Non-whitespace character
			if inWhitespaceSequence {
				inWhitespaceSequence = false
				// If we've already emitted non-space characters, emit a single space
				if emittedAnyNonSpace {
					b.WriteByte(' ')
					normalizedToOriginal = append(normalizedToOriginal, whitespaceSequenceStart)
					normalizedByteIndex++
				}
			}

			// Emit the non-whitespace rune and map its bytes
			encodedRune := make([]byte, size)
			utf8.EncodeRune(encodedRune, r)
			b.Write(encodedRune)

			for j := range size {
				originalToNormalized[originalByteIndex+j] = normalizedByteIndex + j
				normalizedToOriginal = append(normalizedToOriginal, originalByteIndex)
			}
			normalizedByteIndex += size
			emittedAnyNonSpace = true
		}
		originalByteIndex += size
	}

	return b.String(), normalizedToOriginal, originalToNormalized
}
