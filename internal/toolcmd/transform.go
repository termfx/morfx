package toolcmd

import (
	"os"
	"strings"
)

// WriteModifiedSource persists modified source back to disk when the command
// is operating on a file and the transformed content actually changed.
func WriteModifiedSource(path string, fromFile bool, original, modified string, perm os.FileMode) (bool, error) {
	if !fromFile || strings.TrimSpace(modified) == "" || modified == original {
		return false, nil
	}

	if err := os.WriteFile(path, []byte(modified), perm); err != nil {
		return false, err
	}

	return true, nil
}
