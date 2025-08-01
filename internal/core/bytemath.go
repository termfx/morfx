package core

import (
	"github.com/garaekz/fileman/internal/matcher"
	"github.com/garaekz/fileman/internal/model"
)

// getCached returns a (possibly cached) matcher for the rule.
// If not present it compiles/creates and stores atomically.
func getCached(cfg *model.Config) (*matcher.Matcher, error) {
	key := cfg.CacheKey()
	mu.RLock()
	if e, ok := data[key]; ok {
		mu.RUnlock()
		return e.mt, e.err
	}
	mu.RUnlock()

	// slow path – build matcher
	mt, err := matcher.New(cfg)
	mu.Lock()
	if err != nil {
		data[key] = &entry{mt: nil, err: err}
	} else {
		data[key] = &entry{mt: mt, err: err}
	}
	mu.Unlock()
	return data[key].mt, err
}
