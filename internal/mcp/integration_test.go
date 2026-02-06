package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestIntegration_FullSession(t *testing.T) {
	// Setup: Load real patterns
	loader := knowledge.NewLoader("../../patterns")
	patterns, err := loader.LoadAll()
	if err != nil {
		t.Skipf("skipping integration test: %v", err)
	}

	idx := knowledge.NewIndex()
	idx.Build(patterns)
	srv := NewServer(idx)

	// Build input: initialize -> initialized -> tools/list -> tools/call
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"tmkb_query","arguments":{"context":"background job processing","language":"python"}}}
`

	var output bytes.Buffer
	err = srv.ServeStdio(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Parse responses
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(lines))
	}

	// Verify initialize response
	var initResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("failed to parse init response: %v", err)
	}
	idFloat, ok := initResp.ID.(float64)
	if !ok || idFloat != 1 {
		t.Errorf("expected id 1, got %v", initResp.ID)
	}

	// Verify tools/list response
	var listResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	idFloat, ok = listResp.ID.(float64)
	if !ok || idFloat != 2 {
		t.Errorf("expected id 2, got %v", listResp.ID)
	}

	// Verify tools/call response
	var callResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[2]), &callResp); err != nil {
		t.Fatalf("failed to parse call response: %v", err)
	}
	idFloat, ok = callResp.ID.(float64)
	if !ok || idFloat != 3 {
		t.Errorf("expected id 3, got %v", callResp.ID)
	}

	result := callResp.Result.(map[string]interface{})
	if result["isError"] != false {
		t.Error("expected successful tool call")
	}
}

func TestIntegration_ErrorRecovery(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	// Invalid request followed by valid request
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
`

	var output bytes.Buffer
	err := srv.ServeStdio(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}

	// First response should be error
	var errResp JSONRPCErrorResponse
	if err := json.Unmarshal([]byte(lines[0]), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("expected code %d, got %d", ErrCodeInvalidParams, errResp.Error.Code)
	}

	// Second response should be success
	var listResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	idFloat, ok := listResp.ID.(float64)
	if !ok || idFloat != 2 {
		t.Errorf("expected id 2, got %v", listResp.ID)
	}
}

func TestIntegration_ValidationErrors(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	// Tools/call with invalid language
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tmkb_query","arguments":{"context":"test","language":"java"}}}
`

	var output bytes.Buffer
	err := srv.ServeStdio(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	result := resp.Result.(map[string]interface{})
	if result["isError"] != true {
		t.Error("expected tool execution error")
	}

	content := result["content"].([]interface{})
	textContent := content[0].(map[string]interface{})
	text := textContent["text"].(string)

	if !strings.Contains(text, "Invalid language") {
		t.Errorf("expected validation error message, got: %s", text)
	}
}
