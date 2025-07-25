package core

import (
	"errors"
	"testing"
)

func TestCLIError_Error(t *testing.T) {
	err := CLIError{Code: "TEST_CODE", Message: "Test message"}
	if err.Error() != "Test message" {
		t.Errorf("Expected error message 'Test message', got '%s'", err.Error())
	}

	errWithDetail := CLIError{Code: "TEST_CODE", Message: "Test message", Detail: "Some detail"}
	if errWithDetail.Error() != "Test message: Some detail" {
		t.Errorf("Expected error message 'Test message: Some detail', got '%s'", errWithDetail.Error())
	}
}

func TestCLIError_String(t *testing.T) {
	err := CLIError{Code: "TEST_CODE", Message: "Test message"}
	if err.String() != "Test message" {
		t.Errorf("Expected string 'Test message', got '%s'", err.String())
	}

	errWithDetail := CLIError{Code: "TEST_CODE", Message: "Test message", Detail: "Some detail"}
	if errWithDetail.String() != "Test message: Some detail" {
		t.Errorf("Expected string 'Test message: Some detail', got '%s'", errWithDetail.String())
	}
}

func TestCLIError_JSON(t *testing.T) {
	err := CLIError{Code: "TEST_CODE", Message: "Test message", Detail: "Some detail"}
	expected := `{"code":"TEST_CODE","message":"Test message","detail":"Some detail"}`
	if err.JSON() != expected {
		t.Errorf("Expected JSON '%s', got '%s'", expected, err.JSON())
	}

	errNoDetail := CLIError{Code: "TEST_CODE_2", Message: "Another message"}
	expectedNoDetail := `{"code":"TEST_CODE_2","message":"Another message"}`
	if errNoDetail.JSON() != expectedNoDetail {
		t.Errorf("Expected JSON '%s', got '%s'", expectedNoDetail, errNoDetail.JSON())
	}
}

func TestWrap(t *testing.T) {
	innerErr := errors.New("inner error")
	wrappedErr := Wrap(ErrIO, "context message", innerErr)

	cliErr, ok := wrappedErr.(CLIError)
	if !ok {
		t.Fatalf("Expected CLIError, got %T", wrappedErr)
	}

	if cliErr.Code != ErrIO {
		t.Errorf("Expected error code %s, got %s", ErrIO, cliErr.Code)
	}
	if cliErr.Message != "context message: inner error" {
		t.Errorf("Expected message 'context message: inner error', got '%s'", cliErr.Message)
	}
}
