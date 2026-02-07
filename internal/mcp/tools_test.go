package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestHandleToolsList_Success(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	result, err := handleToolsList(srv, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	tools := resultMap["tools"].([]interface{})
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0].(map[string]interface{})
	if tool["name"] != "tmkb_query" {
		t.Errorf("expected tool name tmkb_query, got %v", tool["name"])
	}
}

func TestHandleToolsList_BeforeInit(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	_, err := handleToolsList(srv, nil)
	if err == nil {
		t.Fatal("expected error when not initialized")
	}
}

func TestHandleToolsCall_Success(t *testing.T) {
	// Create index with test patterns
	loader := knowledge.NewLoader("../../patterns")
	patterns, _ := loader.LoadAll()
	idx := knowledge.NewIndex()
	idx.Build(patterns)

	srv := NewServer(idx)
	srv.setState(stateInitialized)

	params := map[string]interface{}{
		"name": "tmkb_query",
		"arguments": map[string]interface{}{
			"context":  "background job processing",
			"language": "python",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := handleToolsCall(srv, paramsJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["isError"] != false {
		t.Error("expected isError false")
	}

	content := resultMap["content"].([]interface{})
	if len(content) == 0 {
		t.Error("expected content")
	}
}

func TestHandleToolsCall_UnknownTool(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	params := map[string]interface{}{
		"name":      "unknown_tool",
		"arguments": map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	_, err := handleToolsCall(srv, paramsJSON)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestHandleToolsCall_MissingContext(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	params := map[string]interface{}{
		"name":      "tmkb_query",
		"arguments": map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := handleToolsCall(srv, paramsJSON)
	if err != nil {
		t.Fatalf("expected no protocol error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["isError"] != true {
		t.Error("expected isError true for missing context")
	}
}

func TestHandleToolsCall_InvalidLanguage(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	params := map[string]interface{}{
		"name": "tmkb_query",
		"arguments": map[string]interface{}{
			"context":  "test",
			"language": "java",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := handleToolsCall(srv, paramsJSON)
	if err != nil {
		t.Fatalf("expected no protocol error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["isError"] != true {
		t.Error("expected isError true for invalid language")
	}

	content := resultMap["content"].([]interface{})
	textContent := content[0].(map[string]interface{})
	text := textContent["text"].(string)
	if !contains(text, "Invalid language") {
		t.Errorf("expected error message about invalid language, got %s", text)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
