package golang_test

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/garaekz/fileman/internal/lang/golang"
)

const (
	dslListFile = "testdata/queries/dsl.txt"
	snapDir     = "testdata/queries"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeDSL(dsl string) string {
	sanitized := nonAlphanumeric.ReplaceAllString(dsl, "_")
	if len(sanitized) > 50 {
		hash := sha1.Sum([]byte(dsl))
		return sanitized[:50] + "_" + hex.EncodeToString(hash[:4])
	}
	return sanitized
}

func TestDSLQuerySnapshots(t *testing.T) {
	updateSnapshots := os.Getenv("SNAP_UPDATE") == "1"

	dslFile, err := os.Open(dslListFile)
	require.NoError(t, err, "failed to open dsl list file")
	defer dslFile.Close()

	p := golang.New()
	scanner := bufio.NewScanner(dslFile)
	var updatedCount int

	for scanner.Scan() {
		dsl := strings.TrimSpace(scanner.Text())
		if dsl == "" || strings.HasPrefix(dsl, "#") {
			continue
		}

		t.Run(dsl, func(t *testing.T) {
			query, err := p.TranslateDSL(dsl)
			require.NoError(t, err, "TranslateDSL failed for: %s", dsl)
			require.NotEmpty(t, query, "TranslateDSL returned empty query for: %s", dsl)

			snapFileName := sanitizeDSL(dsl) + ".snap"
			snapFilePath := filepath.Join(snapDir, snapFileName)

			if updateSnapshots {
				err := os.WriteFile(snapFilePath, []byte(query+"\n"), 0o644)
				require.NoError(t, err, "failed to write snapshot file: %s", snapFilePath)
				updatedCount++
			} else {
				expectedQueryBytes, err := os.ReadFile(snapFilePath)
				if os.IsNotExist(err) {
					t.Fatalf("snapshot not found for DSL: '%s'. Run with SNAP_UPDATE=1 to generate it.", dsl)
				}
				require.NoError(t, err, "failed to read snapshot file: %s", snapFilePath)

				expectedQuery := string(expectedQueryBytes)
				// Normalize newlines for comparison
				normalizedExpected := strings.ReplaceAll(expectedQuery, "\r\n", "\n")
				normalizedActual := strings.ReplaceAll(query+"\n", "\r\n", "\n")

				assert.Equal(t, normalizedExpected, normalizedActual, "snapshot mismatch for DSL: %s", dsl)
			}
		})
	}

	require.NoError(t, scanner.Err(), "failed to scan dsl list file")

	if updateSnapshots && updatedCount > 0 {
		fmt.Printf("Updated %d snapshots in %s\n", updatedCount, snapDir)
	}
}
