package core

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCLIError_JSON(t *testing.T) {
	err := Wrap(ErrInvalidRegex, "bad", os.ErrInvalid)
	ce, ok := err.(CLIError)
	if !ok {
		t.Fatalf("wrap did not return CLIError")
	}
	raw := ce.JSON()
	var decoded map[string]string
	if json.Unmarshal([]byte(raw), &decoded) != nil {
		t.Fatalf("json unmarshal failed")
	}
	if decoded["code"] != ErrInvalidRegex {
		t.Fatalf("wrong code json: %v", decoded)
	}
}
