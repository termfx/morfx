package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("ses_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("ses_%s", hex.EncodeToString(bytes))
}

// generateID creates a unique identifier with a prefix
func generateID(prefix string) string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes))
}
