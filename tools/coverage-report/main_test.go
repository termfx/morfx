package main

import (
	"strings"
	"testing"
)

func TestGenerateMarkdownReport(t *testing.T) {
	report := generateMarkdownReport()

	// Test basic structure - simplified checks
	tests := []struct {
		name     string
		contains string
		required bool
	}{
		{"has title", "# Code Coverage Report", true},
		{"has generation info", "Generated:", true},
		{"has target section", "Target:", true},
		{"has core logic", "Core Logic", true},
		{"has mcp protocol", "MCP Protocol", true},
		{"has providers", "Providers", true},
		{"has database", "Database", true},
		{"has cli", "CLI", true},
		{"has coverage commands", "Coverage Commands", true},
		{"has make test", "make test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(report, tt.contains) {
				t.Errorf("Report missing required content: %s", tt.contains)
			}
		})
	}
}

func TestReportContent(t *testing.T) {
	report := generateMarkdownReport()

	// Basic sanity checks
	if len(report) < 500 {
		t.Error("Report seems too short")
	}

	if !strings.Contains(report, "morfx") {
		t.Error("Report should mention project name")
	}
}

func TestReportFormatValidation(t *testing.T) {
	report := generateMarkdownReport()

	// Check markdown structure
	if strings.Count(report, "#") < 3 {
		t.Error("Report should have multiple sections")
	}

	if strings.Count(report, "|") < 10 {
		t.Error("Report should have table structure")
	}
}

func TestIntegrationWithCoverageCheck(t *testing.T) {
	// Just test that the report generation doesn't crash
	report := generateMarkdownReport()

	if report == "" {
		t.Error("Report generation failed")
	}
}
