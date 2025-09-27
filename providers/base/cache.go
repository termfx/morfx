package base

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	sitter "github.com/smacker/go-tree-sitter"
)

// ASTCache is a lock-free cache for parsed ASTs (shared across all providers)
type ASTCache struct {
	cache       sync.Map // Lock-free concurrent map
	hits        atomic.Int64
	misses      atomic.Int64
	evictions   atomic.Int64
	maxAge      time.Duration
	cleanupOnce sync.Once // Ensures only one cleanup goroutine runs
}

// CachedAST holds parsed tree with metadata
type CachedAST struct {
	tree      *sitter.Tree
	source    []byte
	hash      string
	timestamp time.Time
	hitCount  atomic.Int32
}

// GlobalCache is the singleton cache instance shared across all providers
var GlobalCache = &ASTCache{
	maxAge: 5 * time.Minute,
}

// GetOrParse returns cached AST or parses new one
func (c *ASTCache) GetOrParse(parser *sitter.Parser, source []byte) (*sitter.Tree, bool) {
	// Calculate hash
	hash := c.hash(source)

	// Try cache first (lock-free read)
	if cached, ok := c.cache.Load(hash); ok {
		c.hits.Add(1)
		ast := cached.(*CachedAST)
		ast.hitCount.Add(1)

		// Check if expired
		if time.Since(ast.timestamp) > c.maxAge {
			c.cache.Delete(hash)
			c.evictions.Add(1)
			ast.tree.Close()
		} else {
			return ast.tree.Copy(), true
		}
	}

	c.misses.Add(1)

	// Parse new tree using ParseCtx
	tree, err := parser.ParseCtx(context.TODO(), nil, source)
	if err != nil || tree == nil {
		return nil, false
	}

	// Store in cache (lock-free write)
	cachedAST := &CachedAST{
		tree:      tree.Copy(),
		source:    source,
		hash:      hash,
		timestamp: time.Now(),
	}

	if _, loaded := c.cache.LoadOrStore(hash, cachedAST); loaded {
		// Another goroutine populated the cache first. Release our stored copy
		// and continue returning the freshly parsed tree to the caller.
		cachedAST.tree.Close()
		return tree, false
	}

	// Start single cleanup goroutine on first miss
	c.cleanupOnce.Do(func() {
		go c.cleanupOldEntries()
	})

	return tree, false
}

// hash generates SHA256 for source
func (c *ASTCache) hash(source []byte) string {
	hash := sha256.Sum256(source)
	return hex.EncodeToString(hash[:])
}

// cleanupOldEntries removes expired entries
func (c *ASTCache) cleanupOldEntries() {
	interval := c.maxAge
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		c.pruneExpired()
		<-ticker.C
	}
}

func (c *ASTCache) pruneExpired() {
	now := time.Now()
	c.cache.Range(func(key, value any) bool {
		ast := value.(*CachedAST)
		if now.Sub(ast.timestamp) > c.maxAge {
			c.cache.Delete(key)
			ast.tree.Close()
			c.evictions.Add(1)
		}
		return true
	})
}

// Stats returns cache statistics
func (c *ASTCache) Stats() map[string]int64 {
	return map[string]int64{
		"hits":      c.hits.Load(),
		"misses":    c.misses.Load(),
		"evictions": c.evictions.Load(),
		"hit_rate":  c.hits.Load() * 100 / (c.hits.Load() + c.misses.Load() + 1),
	}
}
