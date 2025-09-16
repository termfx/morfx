package tools

import (
	"testing"
)

// extractContentText is a helper to extract text from the content field which can be either
// a map[string]any with a text field or an array of such maps
func extractContentText(t *testing.T, result any) string {
	t.Helper()

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("Result should be a map")
		return ""
	}

	// content can be either []map[string]any or []interface{}
	if contentArray, ok := resultMap["content"].([]map[string]any); ok && len(contentArray) > 0 {
		if text, ok := contentArray[0]["text"].(string); ok {
			return text
		}
	} else if contentInterface, ok := resultMap["content"].([]any); ok && len(contentInterface) > 0 {
		if contentItem, ok := contentInterface[0].(map[string]any); ok {
			if text, ok := contentItem["text"].(string); ok {
				return text
			}
		}
	}

	t.Error("Could not extract text from content")
	return ""
}

// hasContentArray checks if result has a content array field
func hasContentArray(result any) bool {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return false
	}

	_, hasArray := resultMap["content"].([]map[string]any)
	if !hasArray {
		_, hasArray = resultMap["content"].([]any)
	}

	return hasArray
}

// Fix for old tests that expect content to be a map
// This converts the new format to the old format for compatibility
func convertContentToMap(result any) (map[string]any, bool) {
	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, false
	}

	// Try to get content array and convert to map
	contentMap := make(map[string]any)

	if contentArray, ok := resultMap["content"].([]map[string]any); ok && len(contentArray) > 0 {
		// Use the first item as the content map
		contentMap = contentArray[0]
	} else if contentInterface, ok := resultMap["content"].([]any); ok && len(contentInterface) > 0 {
		if item, ok := contentInterface[0].(map[string]any); ok {
			contentMap = item
		}
	}

	// Update the result to use the map format
	resultMap["content"] = contentMap
	return contentMap, true
}
