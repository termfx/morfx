package model

import "errors"

// Sentinel errors for programmatic checking.
var (
	ErrNoMatchesFound        = errors.New("no matches found")
	ErrMustMatchFailed       = errors.New("must_match not satisfied")
	ErrMustChangeBytesFailed = errors.New("must_change_bytes not satisfied")
	ErrWriteRace             = errors.New("file changed on disk during operation")
	ErrInvalidRegex          = errors.New("invalid regex")
)

// ErrorCode provides a machine-readable error type for JSON output.
type ErrorCode string

const (
	ECNone                  ErrorCode = ""
	ECNoMatch               ErrorCode = "ERR_NO_MATCH"
	ECMustMatchFailed       ErrorCode = "ERR_MUST_MATCH"
	ECMustChangeBytesFailed ErrorCode = "ERR_MUST_CHANGE_BYTES"
	ECWriteRace             ErrorCode = "ERR_WRITE_RACE"
	ECInvalidRegex          ErrorCode = "ERR_INVALID_REGEX"
	ECReadError             ErrorCode = "ERR_READ_FILE"
	ECWriteError            ErrorCode = "ERR_WRITE_FILE"
	ECConfigError           ErrorCode = "ERR_CONFIG"
	ECUnknown               ErrorCode = "ERR_UNKNOWN"
)
