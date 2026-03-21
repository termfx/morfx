package toolenv

import (
	"encoding/json"
	"fmt"
	"io"
)

// ReadJSON decodes all data from the provided reader into the supplied generic type.
func ReadJSON[T any](r io.Reader) (*T, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	var payload T
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode json input: %w", err)
	}
	return &payload, nil
}

// WriteJSON encodes the supplied value as compact JSON to the writer.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode json output: %w", err)
	}
	return nil
}

// WriteError emits a standard error envelope for CLI tools.
func WriteError(w io.Writer, message string, err error) error {
	payload := map[string]any{
		"error": map[string]any{
			"message": message,
		},
	}
	if err != nil {
		payload["error"].(map[string]any)["details"] = err.Error()
	}
	return WriteJSON(w, payload)
}
