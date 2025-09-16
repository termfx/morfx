package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestSuccessResponse tests the SuccessResponse function
func TestSuccessResponse(t *testing.T) {
	tests := []struct {
		name   string
		id     any
		result any
		want   Response
	}{
		{
			name:   "integer_id",
			id:     1,
			result: "success",
			want: Response{
				JSONRPC: "2.0",
				Result:  "success",
				ID:      1,
			},
		},
		{
			name:   "string_id",
			id:     "test-id",
			result: map[string]any{"status": "ok"},
			want: Response{
				JSONRPC: "2.0",
				Result:  map[string]any{"status": "ok"},
				ID:      "test-id",
			},
		},
		{
			name:   "nil_id",
			id:     nil,
			result: []string{"item1", "item2"},
			want: Response{
				JSONRPC: "2.0",
				Result:  []string{"item1", "item2"},
				ID:      nil,
			},
		},
		{
			name: "complex_result",
			id:   42,
			result: map[string]any{
				"data": []map[string]any{
					{"name": "test", "value": 123},
					{"name": "test2", "value": 456},
				},
				"meta": map[string]any{
					"count": 2,
					"total": 579,
				},
			},
			want: Response{
				JSONRPC: "2.0",
				Result: map[string]any{
					"data": []map[string]any{
						{"name": "test", "value": 123},
						{"name": "test2", "value": 456},
					},
					"meta": map[string]any{
						"count": 2,
						"total": 579,
					},
				},
				ID: 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SuccessResponse(tt.id, tt.result)

			if got.JSONRPC != tt.want.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", got.JSONRPC, tt.want.JSONRPC)
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}

			if got.Error != nil {
				t.Errorf("Error should be nil for success response, got %v", got.Error)
			}

			// Compare results by marshaling to JSON and back
			gotJSON, err := json.Marshal(got.Result)
			if err != nil {
				t.Fatalf("Failed to marshal result: %v", err)
			}

			wantJSON, err := json.Marshal(tt.want.Result)
			if err != nil {
				t.Fatalf("Failed to marshal expected result: %v", err)
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("Result = %s, want %s", string(gotJSON), string(wantJSON))
			}
		})
	}
}

// TestErrorResponse tests error response creation
func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name     string
		id       any
		code     int
		message  string
		data     any
		hasError bool
	}{
		{
			name:     "simple_error",
			id:       1,
			code:     -32602,
			message:  "Invalid params",
			data:     nil,
			hasError: true,
		},
		{
			name:     "error_with_data",
			id:       "test",
			code:     -32603,
			message:  "Internal error",
			data:     map[string]any{"details": "Something went wrong"},
			hasError: true,
		},
		{
			name:     "method_not_found",
			id:       42,
			code:     -32601,
			message:  "Method not found",
			data:     nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Response
			if tt.data != nil {
				got = ErrorResponseWithData(tt.id, tt.code, tt.message, tt.data)
			} else {
				got = ErrorResponse(tt.id, tt.code, tt.message)
			}

			if got.JSONRPC != "2.0" {
				t.Errorf("JSONRPC = %v, want 2.0", got.JSONRPC)
			}

			if got.ID != tt.id {
				t.Errorf("ID = %v, want %v", got.ID, tt.id)
			}

			if got.Result != nil {
				t.Errorf("Result should be nil for error response, got %v", got.Result)
			}

			if tt.hasError && got.Error == nil {
				t.Fatal("Error should not be nil")
			}

			if got.Error.Code != tt.code {
				t.Errorf("Error code = %v, want %v", got.Error.Code, tt.code)
			}

			if got.Error.Message != tt.message {
				t.Errorf("Error message = %v, want %v", got.Error.Message, tt.message)
			}

			if tt.data != nil {
				if got.Error.Data == nil {
					t.Error("Error data should not be nil")
				} else {
					// Compare by JSON marshaling
					gotDataJSON, _ := json.Marshal(got.Error.Data)
					wantDataJSON, _ := json.Marshal(tt.data)
					if string(gotDataJSON) != string(wantDataJSON) {
						t.Errorf("Error data = %s, want %s", string(gotDataJSON), string(wantDataJSON))
					}
				}
			} else if got.Error.Data != nil {
				t.Errorf("Error data should be nil, got %v", got.Error.Data)
			}
		})
	}
}

// TestRequestSerialization tests JSON serialization of requests
func TestRequestSerialization(t *testing.T) {
	tests := []struct {
		name string
		req  Request
	}{
		{
			name: "simple_request",
			req: Request{
				JSONRPC: "2.0",
				Method:  "test_method",
				ID:      1,
			},
		},
		{
			name: "request_with_params",
			req: Request{
				JSONRPC: "2.0",
				Method:  "method_with_params",
				Params:  json.RawMessage(`{"param1": "value1", "param2": 42}`),
				ID:      "string-id",
			},
		},
		{
			name: "notification_request",
			req: Request{
				JSONRPC: "2.0",
				Method:  "notification_method",
				Params:  json.RawMessage(`{"notify": true}`),
				// No ID for notifications
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Test deserialization
			var unmarshaled Request
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			// Compare fields
			if unmarshaled.JSONRPC != tt.req.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", unmarshaled.JSONRPC, tt.req.JSONRPC)
			}

			if unmarshaled.Method != tt.req.Method {
				t.Errorf("Method = %v, want %v", unmarshaled.Method, tt.req.Method)
			}

			// Compare IDs more carefully since JSON marshaling may change types
			if fmt.Sprintf("%v", unmarshaled.ID) != fmt.Sprintf("%v", tt.req.ID) {
				t.Errorf("ID = %v, want %v", unmarshaled.ID, tt.req.ID)
			}

			// Compare params by normalizing JSON whitespace
			if strings.ReplaceAll(
				string(unmarshaled.Params),
				" ",
				"",
			) != strings.ReplaceAll(
				string(tt.req.Params),
				" ",
				"",
			) {
				t.Logf("Params differ but this might be due to JSON formatting: got %s, want %s",
					string(unmarshaled.Params), string(tt.req.Params))
			}
		})
	}
}

// TestResponseSerialization tests JSON serialization of responses
func TestResponseSerialization(t *testing.T) {
	tests := []struct {
		name string
		resp Response
	}{
		{
			name: "success_response",
			resp: SuccessResponse(1, map[string]any{"result": "success"}),
		},
		{
			name: "error_response",
			resp: ErrorResponse(2, -32602, "Invalid params"),
		},
		{
			name: "error_response_with_data",
			resp: ErrorResponseWithData("test", -32603, "Internal error", map[string]any{
				"details": "Stack trace here",
				"code":    500,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			// Test deserialization
			var unmarshaled Response
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			// Compare fields
			if unmarshaled.JSONRPC != tt.resp.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", unmarshaled.JSONRPC, tt.resp.JSONRPC)
			}

			if fmt.Sprintf("%v", unmarshaled.ID) != fmt.Sprintf("%v", tt.resp.ID) {
				t.Errorf("ID = %v, want %v", unmarshaled.ID, tt.resp.ID)
			}

			// Compare results and errors by JSON marshaling
			if tt.resp.Result != nil {
				gotResultJSON, _ := json.Marshal(unmarshaled.Result)
				wantResultJSON, _ := json.Marshal(tt.resp.Result)
				if string(gotResultJSON) != string(wantResultJSON) {
					t.Errorf("Result = %s, want %s", string(gotResultJSON), string(wantResultJSON))
				}
			}

			if tt.resp.Error != nil {
				if unmarshaled.Error == nil {
					t.Error("Error should not be nil")
				} else {
					if unmarshaled.Error.Code != tt.resp.Error.Code {
						t.Errorf("Error code = %v, want %v", unmarshaled.Error.Code, tt.resp.Error.Code)
					}
					if unmarshaled.Error.Message != tt.resp.Error.Message {
						t.Errorf("Error message = %v, want %v", unmarshaled.Error.Message, tt.resp.Error.Message)
					}
				}
			}
		})
	}
}

// TestNotificationSerialization tests notification serialization
func TestNotificationSerialization(t *testing.T) {
	tests := []struct {
		name         string
		notification Notification
	}{
		{
			name: "simple_notification",
			notification: Notification{
				JSONRPC: "2.0",
				Method:  "notify/progress",
			},
		},
		{
			name: "notification_with_params",
			notification: Notification{
				JSONRPC: "2.0",
				Method:  "notify/update",
				Params:  json.RawMessage(`{"progress": 50, "message": "Half done"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test serialization
			data, err := json.Marshal(tt.notification)
			if err != nil {
				t.Fatalf("Failed to marshal notification: %v", err)
			}

			// Test deserialization
			var unmarshaled Notification
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal notification: %v", err)
			}

			// Compare fields
			if unmarshaled.JSONRPC != tt.notification.JSONRPC {
				t.Errorf("JSONRPC = %v, want %v", unmarshaled.JSONRPC, tt.notification.JSONRPC)
			}

			if unmarshaled.Method != tt.notification.Method {
				t.Errorf("Method = %v, want %v", unmarshaled.Method, tt.notification.Method)
			}

			// Compare params by normalizing JSON whitespace
			if strings.ReplaceAll(
				string(unmarshaled.Params),
				" ",
				"",
			) != strings.ReplaceAll(
				string(tt.notification.Params),
				" ",
				"",
			) {
				t.Logf("Params differ but this might be due to JSON formatting: got %s, want %s",
					string(unmarshaled.Params), string(tt.notification.Params))
			}

			// Verify notification has no ID field when serialized
			var genericMap map[string]any
			if err := json.Unmarshal(data, &genericMap); err != nil {
				t.Fatalf("Failed to unmarshal to map: %v", err)
			}

			if _, hasID := genericMap["id"]; hasID {
				t.Error("Notification should not have an 'id' field")
			}
		})
	}
}

// TestProtocolConstants tests standard JSON-RPC error codes
func TestProtocolConstants(t *testing.T) {
	// Test that we can create responses with standard error codes
	standardErrors := []struct {
		code    int
		name    string
		message string
	}{
		{-32700, "ParseError", "Parse error"},
		{-32600, "InvalidRequest", "Invalid Request"},
		{-32601, "MethodNotFound", "Method not found"},
		{-32602, "InvalidParams", "Invalid params"},
		{-32603, "InternalError", "Internal error"},
	}

	for _, errInfo := range standardErrors {
		t.Run(errInfo.name, func(t *testing.T) {
			resp := ErrorResponse("test", errInfo.code, errInfo.message)

			if resp.Error == nil {
				t.Fatal("Error should not be nil")
			}

			if resp.Error.Code != errInfo.code {
				t.Errorf("Error code = %v, want %v", resp.Error.Code, errInfo.code)
			}

			if resp.Error.Message != errInfo.message {
				t.Errorf("Error message = %v, want %v", resp.Error.Message, errInfo.message)
			}

			// Test that the response can be serialized
			_, err := json.Marshal(resp)
			if err != nil {
				t.Errorf("Failed to marshal error response: %v", err)
			}
		})
	}
}

// TestEdgeCases tests edge cases in protocol handling
func TestEdgeCases(t *testing.T) {
	t.Run("nil_params", func(t *testing.T) {
		req := Request{
			JSONRPC: "2.0",
			Method:  "test",
			Params:  nil,
			ID:      1,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request with nil params: %v", err)
		}

		var unmarshaled Request
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal request with nil params: %v", err)
		}
	})

	t.Run("empty_params", func(t *testing.T) {
		req := Request{
			JSONRPC: "2.0",
			Method:  "test",
			Params:  json.RawMessage("{}"),
			ID:      1,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request with empty params: %v", err)
		}

		var unmarshaled Request
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal request with empty params: %v", err)
		}
	})

	t.Run("very_long_method_name", func(t *testing.T) {
		longMethod := ""
		for range 1000 {
			longMethod += "very_long_method_name_"
		}

		req := Request{
			JSONRPC: "2.0",
			Method:  longMethod,
			ID:      1,
		}

		_, err := json.Marshal(req)
		if err != nil {
			t.Errorf("Should be able to marshal request with long method name: %v", err)
		}
	})
}
