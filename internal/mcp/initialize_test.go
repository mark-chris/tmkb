package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestHandleInitialize_Success(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	params := map[string]interface{}{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "TestClient",
			"version": "1.0.0",
		},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := handleInitialize(srv, paramsJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["protocolVersion"] != "2025-11-25" {
		t.Errorf("expected protocol version 2025-11-25, got %v", resultMap["protocolVersion"])
	}

	if srv.getState() != stateInitializing {
		t.Errorf("expected state Initializing, got %v", srv.getState())
	}
}

func TestHandleInitialize_DuplicateInit(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	params := map[string]interface{}{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	_, err := handleInitialize(srv, paramsJSON)
	if err == nil {
		t.Fatal("expected error for duplicate initialization")
	}
}
