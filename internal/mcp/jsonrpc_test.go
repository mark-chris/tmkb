package mcp

import (
	"testing"
)

func TestParseRequest_ValidRequest(t *testing.T) {
	input := []byte(`{
		"jsonrpc": "2.0",
		"method": "initialize",
		"params": {"clientInfo": {"name": "test"}},
		"id": 1
	}`)

	req, err := parseRequest(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
	}
	if req.Method != "initialize" {
		t.Errorf("expected method initialize, got %s", req.Method)
	}
	if req.ID == nil {
		t.Error("expected ID to be non-nil")
	}
}

func TestParseRequest_MalformedJSON(t *testing.T) {
	input := []byte(`{invalid json}`)

	_, err := parseRequest(input)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestParseRequest_MissingJSONRPC(t *testing.T) {
	input := []byte(`{
		"method": "initialize",
		"id": 1
	}`)

	_, err := parseRequest(input)
	if err == nil {
		t.Fatal("expected error for missing jsonrpc field, got nil")
	}
}

func TestCreateResponse_Success(t *testing.T) {
	result := map[string]string{"status": "ok"}
	id := 123

	resp := createResponse(result, id)

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	if resp.Result == nil {
		t.Error("expected result to be non-nil")
	}
	if resp.ID != id {
		t.Errorf("expected id %v, got %v", id, resp.ID)
	}
}

func TestCreateErrorResponse_ProtocolError(t *testing.T) {
	code := -32600
	message := "Invalid Request"
	id := 456

	resp := createErrorResponse(code, message, nil, id)

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	if resp.Error.Code != code {
		t.Errorf("expected error code %d, got %d", code, resp.Error.Code)
	}
	if resp.Error.Message != message {
		t.Errorf("expected error message %s, got %s", message, resp.Error.Message)
	}
	if resp.ID != id {
		t.Errorf("expected id %v, got %v", id, resp.ID)
	}
}
