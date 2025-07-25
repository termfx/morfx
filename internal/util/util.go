package util

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"

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
	b.Grow(
		len(s),
	) // la normalizada nunca será más larga que la original en bytes si no hay RuneError

	normalizedToOriginal = make([]int, 0, len(s))
	originalToNormalized = make([]int, len(s))
	for i := range originalToNormalized {
		originalToNormalized[i] = -1
	}

	emittedAny := false // ya emitimos al menos un no-espacio
	inWS := false       // estamos dentro de una secuencia de whitespace
	wsStart := 0        // índice byte del primer whitespace de la secuencia
	normIdx := 0        // índice byte actual en la cadena normalizada

	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// String con byte inválido. Lo tratamos como "no whitespace" para preservar el byte.
			// Se puede ajustar política según necesidad.
		}

		if unicode.IsSpace(r) {
			if !inWS {
				inWS = true
				wsStart = i
			}
			// No asignamos originalToNormalized para estos bytes (permanecerán -1)
			i += size
			continue
		}

		// r no es whitespace
		if inWS {
			// Acabamos de cerrar una secuencia de whitespace
			if emittedAny {
				// Emitimos un solo espacio representando toda la secuencia previa
				b.WriteByte(' ')
				normalizedToOriginal = append(
					normalizedToOriginal,
					wsStart,
				) // convención: mapea al primer byte del span
				normIdx++ // 1 byte
			}
			inWS = false
		}

		// Emitimos el rune tal cual (en bytes) y mapeamos por byte
		var buf [utf8.UTFMax]byte
		n := utf8.EncodeRune(buf[:], r)
		// En teoría n == size casi siempre para UTF-8 válido. Si no, igual es seguro.

		// Mapear original -> normalizado (por cada byte del rune original)
		for j := range size {
			if j < n {
				originalToNormalized[i+j] = normIdx + j
			} else {
				// rune raro (RuneError 3 bytes vs size 1, etc.) mapeamos al primer byte normalizado
				originalToNormalized[i+j] = normIdx
			}
		}

		// Mapear normalizado -> original
		for range n {
			normalizedToOriginal = append(
				normalizedToOriginal,
				i,
			) // apúntalo al inicio del rune original
		}

		b.Write(buf[:n])
		normIdx += n
		emittedAny = true
		i += size
	}

	// Si terminamos en whitespace, no emitimos el espacio (evitamos trailing). Nada que hacer.

	return b.String(), normalizedToOriginal, originalToNormalized
}

// --- Filesystem Helpers ---

// WriteFileAtomic writes data to a file atomically.
func WriteFileAtomic(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode()
	}
	dir := filepath.Dir(path)
	// Use a more descriptive temp file pattern
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // Clean up on error
	defer func() { _ = tmp.Close() }()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// RaceDetected checks if a file was modified on disk between reading and writing.
func RaceDetected(before, after os.FileInfo) bool {
	if before == nil || after == nil {
		return false
	}
	// Also check size, as some filesystems have low-resolution timestamps.
	return !before.ModTime().Equal(after.ModTime()) || before.Size() != after.Size()
}

// ExpandGlobs expands a list of file paths, including glob patterns.
func ExpandGlobs(files []string) []string {
	var out []string
	for _, f := range files {
		if f == "-" {
			out = append(out, f)
			continue
		}
		if strings.ContainsAny(f, "*?[") {
			matches, _ := filepath.Glob(f)
			out = append(out, matches...)
		} else {
			out = append(out, f)
		}
	}
	return out
}

// --- Hashing Helpers ---

// SHA1Hex computes the SHA1 hash of a byte slice and returns it as a hex string.
func SHA1Hex(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}

// SHA1FileHex computes the SHA1 hash of a file's content.
func SHA1FileHex(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return SHA1Hex(b)
}

// EscapeRegexLiteral escapes all regex metacharacters in a string.
func EscapeRegexLiteral(s string) string {
	return regexp.QuoteMeta(s)
}

// --- Diff Helpers ---

const (
	colorReset = "\x1b[0m"
	colorRed   = "\x1b[31m"
	colorGreen = "\x1b[32m"
	colorCyan  = "\x1b[36m"
)

// UnifiedDiff generates a colored or plain unified diff string.
func UnifiedDiff(orig, mod, filename string, context int, color bool) string {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(orig),
		B:        difflib.SplitLines(mod),
		FromFile: filename,
		ToFile:   filename + " (modified)",
		Context:  context,
	}
	text, err := difflib.GetUnifiedDiffString(d)
	if err != nil {
		return "(diff error: " + err.Error() + ")"
	}

	if !color {
		return text
	}

	var sb strings.Builder
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if i == len(lines)-1 && l == "" {
			continue // Skip trailing newline from split
		}
		switch {
		case strings.HasPrefix(l, "+"):
			sb.WriteString(colorGreen + l + colorReset + "\n")
		case strings.HasPrefix(l, "-"):
			sb.WriteString(colorRed + l + colorReset + "\n")
		case strings.HasPrefix(l, "@"):
			sb.WriteString(colorCyan + l + colorReset + "\n")
		default:
			sb.WriteString(l + "\n")
		}
	}
	return sb.String()
}
