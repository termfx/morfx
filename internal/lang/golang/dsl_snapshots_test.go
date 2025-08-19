package golang

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/termfx/morfx/internal/parser"
)

func TestDSLQuerySnapshots(t *testing.T) {
	provider := NewProvider()
	uniParser := parser.NewUniversalParser()

	// Read DSL queries from testdata
	dslFile := filepath.Join("testdata", "queries", "dsl.txt")
	file, err := os.Open(dslFile)
	if err != nil {
		t.Fatalf("Failed to open DSL file: %v", err)
	}
	defer file.Close()

	updateSnapshots := os.Getenv("SNAP_UPDATE") == "1"
	if updateSnapshots {
		t.Log("SNAP_UPDATE=1: Updating snapshots")
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		t.Run(line, func(t *testing.T) {
			// Parse the DSL query
			query, err := uniParser.ParseQuery(line)
			if err != nil {
				t.Fatalf("Failed to parse DSL %q: %v", line, err)
			}

			// Translate to Tree-sitter query
			tsQuery, err := provider.TranslateQuery(query)
			if err != nil {
				t.Fatalf("Failed to translate query %q: %v", line, err)
			}

			// Generate snapshot filename from DSL
			snapName := generateSnapName(line)
			snapFile := filepath.Join("testdata", "queries", snapName+".snap")

			if updateSnapshots {
				// Update the snapshot file
				err := os.WriteFile(snapFile, []byte(tsQuery+"\n"), 0o644)
				if err != nil {
					t.Fatalf("Failed to write snapshot file %s: %v", snapFile, err)
				}
				t.Logf("Updated snapshot: %s", snapFile)
			} else {
				// Compare against existing snapshot
				expected, err := os.ReadFile(snapFile)
				if err != nil {
					t.Fatalf("Failed to read snapshot file %s: %v (run 'make regen-snapshots' to generate)", snapFile, err)
				}

				expectedStr := strings.TrimSpace(string(expected))
				actualStr := strings.TrimSpace(tsQuery)

				if expectedStr != actualStr {
					t.Errorf("Snapshot mismatch for query %q\nExpected:\n%s\nActual:\n%s\nFile: %s",
						line, expectedStr, actualStr, snapFile)
				}
			}
		})
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading DSL file: %v", err)
	}
}

// generateSnapName creates a filename from a DSL query
func generateSnapName(dsl string) string {
	// Replace special characters with underscores
	name := strings.ReplaceAll(dsl, ":", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Remove consecutive underscores
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// Remove leading/trailing underscores
	name = strings.Trim(name, "_")

	return name
}
