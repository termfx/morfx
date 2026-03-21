package mcp

import (
	"strings"
	"testing"
)

// TestGenerateSessionID tests session ID generation
func TestGenerateSessionID(t *testing.T) {
	id1 := generateSessionID()
	id2 := generateSessionID()

	// IDs should be non-empty
	if id1 == "" {
		t.Error("Generated session ID should not be empty")
	}

	if id2 == "" {
		t.Error("Generated session ID should not be empty")
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Generated session IDs should be unique")
	}

	// ID should have reasonable length (UUIDs are typically 36 characters)
	if len(id1) < 10 {
		t.Errorf("Session ID seems too short: %s", id1)
	}

	if len(id2) < 10 {
		t.Errorf("Session ID seems too short: %s", id2)
	}
}

// TestGenerateID tests general ID generation
func TestGenerateID(t *testing.T) {
	id1 := generateID("test")
	id2 := generateID("test")

	// IDs should be non-empty
	if id1 == "" {
		t.Error("Generated ID should not be empty")
	}

	if id2 == "" {
		t.Error("Generated ID should not be empty")
	}

	// IDs should be unique
	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	// ID should have reasonable length
	if len(id1) < 8 {
		t.Errorf("Generated ID seems too short: %s", id1)
	}

	// Should contain prefix
	if !strings.HasPrefix(id1, "test_") {
		t.Errorf("Generated ID should have prefix 'test_': %s", id1)
	}
}

// TestGenerateIDFormat tests ID format
func TestGenerateIDFormat(t *testing.T) {
	id := generateID("format")

	// Should not contain spaces or special characters that might cause issues
	if strings.Contains(id, " ") {
		t.Error("Generated ID should not contain spaces")
	}

	if strings.Contains(id, "\n") {
		t.Error("Generated ID should not contain newlines")
	}

	if strings.Contains(id, "\t") {
		t.Error("Generated ID should not contain tabs")
	}

	// Should contain the prefix
	if !strings.HasPrefix(id, "format_") {
		t.Errorf("Generated ID should have prefix 'format_': %s", id)
	}
}

// TestMultipleIDGeneration tests generating many IDs for uniqueness
func TestMultipleIDGeneration(t *testing.T) {
	const numIDs = 100
	ids := make(map[string]bool)

	for range numIDs {
		id := generateID("multi")

		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}

		ids[id] = true
	}

	if len(ids) != numIDs {
		t.Errorf("Expected %d unique IDs, got %d", numIDs, len(ids))
	}
}

// TestMultipleSessionIDGeneration tests generating many session IDs for uniqueness
func TestMultipleSessionIDGeneration(t *testing.T) {
	const numIDs = 50
	ids := make(map[string]bool)

	for range numIDs {
		id := generateSessionID()

		if ids[id] {
			t.Errorf("Duplicate session ID generated: %s", id)
		}

		ids[id] = true
	}

	if len(ids) != numIDs {
		t.Errorf("Expected %d unique session IDs, got %d", numIDs, len(ids))
	}
}

// TestIDGenerationPerformance tests that ID generation is reasonably fast
func TestIDGenerationPerformance(t *testing.T) {
	const numGenerations = 1000

	// Test generateID performance
	for range numGenerations {
		id := generateID("perf")
		if id == "" {
			t.Error("Generated ID should not be empty")
			break
		}
	}

	// Test generateSessionID performance
	for range numGenerations {
		id := generateSessionID()
		if id == "" {
			t.Error("Generated session ID should not be empty")
			break
		}
	}

	// If we get here without timing out, performance is acceptable
	t.Logf("Successfully generated %d IDs and %d session IDs", numGenerations, numGenerations)
}
