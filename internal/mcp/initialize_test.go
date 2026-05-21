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
		"protocolVersion": "2025-06-18",
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
	if resultMap["protocolVersion"] != "2025-06-18" {
		t.Errorf("expected protocol version 2025-06-18, got %v", resultMap["protocolVersion"])
	}

	if srv.getState() != stateInitializing {
		t.Errorf("expected state Initializing, got %v", srv.getState())
	}
}

func TestHandleInitialize_EchoesSupportedVersion(t *testing.T) {
	// A supported version other than the default must be echoed back verbatim.
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	params := map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := handleInitialize(srv, paramsJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["protocolVersion"] != "2025-03-26" {
		t.Errorf("expected echoed version 2025-03-26, got %v", resultMap["protocolVersion"])
	}
}

func TestHandleInitialize_FallsBackOnUnsupportedVersion(t *testing.T) {
	// An unsupported version must fall back to the default so a compliant
	// client can still complete the handshake (regression test for the
	// hardcoded 2025-11-25 reply that broke MCP negotiation).
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	params := map[string]interface{}{
		"protocolVersion": "2099-01-01",
		"capabilities":    map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := handleInitialize(srv, paramsJSON)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	resultMap := result.(map[string]interface{})
	if resultMap["protocolVersion"] != defaultProtocolVersion {
		t.Errorf("expected fallback to %s, got %v", defaultProtocolVersion, resultMap["protocolVersion"])
	}
}

func TestHandleInitialize_DuplicateInit(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitialized)

	params := map[string]interface{}{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]interface{}{},
	}
	paramsJSON, _ := json.Marshal(params)

	_, err := handleInitialize(srv, paramsJSON)
	if err == nil {
		t.Fatal("expected error for duplicate initialization")
	}
}
