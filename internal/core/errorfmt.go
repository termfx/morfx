package core

import (
	"encoding/json"
)

// ErrCode enumerates common error identifiers.
const (
	ErrInvalidRegex       = "ERR_INVALID_REGEX"
	ErrParseQuery         = "ERR_PARSE_QUERY"
	ErrUnsupportedLang    = "ERR_UNSUPPORTED_LANG"
	ErrIO                 = "ERR_IO"
	ErrInvalidConfig      = "ERR_INVALID_CONFIG"
	ErrInvalidOccurrences = "ERR_INVALID_OCCURRENCES"
	ErrInvalidOperation   = "ERR_INVALID_OPERATION"
)

// CLIError is a uniform error payload for both human and JSON output.
// When printed with %s it returns Message; with %+v it returns JSON.
type CLIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e CLIError) Error() string {
	if e.Detail != "" {
		return e.Message + ": " + e.Detail
	}
	return e.Message
}

func (e CLIError) String() string {
	if e.Detail != "" {
		return e.Message + ": " + e.Detail
	}
	return e.Message
}

func (e CLIError) JSON() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// Wrap helper generates CLIError with code and wraps inner error for detail.
func Wrap(code, msg string, inner error) error {
	return CLIError{Code: code, Message: msg, Detail: inner.Error()}
}
