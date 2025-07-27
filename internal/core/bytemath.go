package core

import (
	"strings"

	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
)

func computeLineIndex(content string) []int {
	idx := []int{0}
	pos := 0
	for {
		i := strings.IndexByte(content[pos:], '\n')
		if i == -1 {
			break
		}
		pos += i + 1
		idx = append(idx, pos)
	}
	return idx
}

func byteToLine(lineIdx []int, pos int) int {
	lo, hi := 0, len(lineIdx)-1
	line := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		if lineIdx[mid] > pos {
			hi = mid - 1
		} else {
			line = mid
			lo = mid + 1
		}
	}
	return line + 1 // 1-based
}

func byteToLineRange(lineIdx []int, start, end int) (int, int) {
	return byteToLine(lineIdx, start), byteToLine(lineIdx, end-1) // end is exclusive
}

// getCached returns a (possibly cached) matcher for the rule.
// If not present it compiles/creates and stores atomically.
func getCached(cfg *model.Config) (matcher.Matcher, error) {
	key := cfg.CacheKey()
	mu.RLock()
	if e, ok := data[key]; ok {
		mu.RUnlock()
		return e.mt, e.err
	}
	mu.RUnlock()

	// slow path – build matcher
	mt, err := buildMatcher(cfg) // existing helper in core
	mu.Lock()
	data[key] = &entry{mt: mt, err: err}
	mu.Unlock()
	return mt, err
}
