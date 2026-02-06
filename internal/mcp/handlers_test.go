package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestHandleMessage_Initialize(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-11-25","capabilities":{}}`),
	}
	reqData, _ := json.Marshal(req)

	respData, err := srv.handleMessage(reqData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// JSON unmarshals numbers as float64
	idFloat, ok := resp.ID.(float64)
	if !ok || idFloat != 1 {
		t.Errorf("expected id 1, got %v", resp.ID)
	}
}

func TestHandleMessage_MethodNotFound(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}
	reqData, _ := json.Marshal(req)

	respData, err := srv.handleMessage(reqData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var resp JSONRPCErrorResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("expected code %d, got %d", ErrCodeMethodNotFound, resp.Error.Code)
	}
}

func TestHandleMessage_InitializedNotification(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)
	srv.setState(stateInitializing)

	notif := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	notifData, _ := json.Marshal(notif)

	respData, err := srv.handleMessage(notifData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Notifications don't get responses
	if len(respData) != 0 {
		t.Error("expected no response for notification")
	}

	if srv.getState() != stateInitialized {
		t.Errorf("expected state Initialized, got %v", srv.getState())
	}
}
