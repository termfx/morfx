package cli

import (
	"os"
	"testing"

	"github.com/garaekz/fileman/internal/model"
)

func TestProcessFileWithRules_ErrorReadingFile(t *testing.T) {
	r := &Runner{}
	_, err := r.processFileWithRules("nonexistent.txt", []model.ModificationConfig{})
	if err == nil {
		t.Error("Expected error reading nonexistent file, but got nil")
	}
}

func TestProcessFileWithRules_InvalidRule(t *testing.T) {
	r := &Runner{}
	file, _ := os.CreateTemp("", "testfile")
	defer os.Remove(file.Name())

	invalidRule := model.ModificationConfig{
		RuleID:  "invalid",
		Pattern: "(", // Invalid regex
	}

	_, err := r.processFileWithRules(file.Name(), []model.ModificationConfig{invalidRule})
	if err == nil {
		t.Error("Expected error with invalid rule, but got nil")
	}
}

func TestProcessFileWithRules_ContractFailure(t *testing.T) {
	r := &Runner{}
	file, _ := os.CreateTemp("", "testfile")
	defer os.Remove(file.Name())

	file.WriteString("some content")

	contractRule := model.ModificationConfig{
		RuleID:      "contract-fail",
		Pattern:     "content",
		Replacement: "new-content",
		MustMatch:   2, // This will fail
	}

	_, err := r.processFileWithRules(file.Name(), []model.ModificationConfig{contractRule})
	if err == nil {
		t.Error("Expected contract failure error, but got nil")
	}
}
