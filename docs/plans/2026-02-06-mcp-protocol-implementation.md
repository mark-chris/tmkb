# MCP Protocol Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement complete JSON-RPC 2.0 protocol support for MCP server enabling Claude Desktop integration

**Architecture:** Three-layer design with transport (stdio), message routing (handler registry), and handler layer (initialize, tools/list, tools/call). State machine enforces initialization lifecycle. Strict validation with actionable errors.

**Tech Stack:** Go 1.25, JSON-RPC 2.0, MCP Protocol 2025-11-25

---

## Task 1: JSON-RPC Message Types and Helpers

**Files:**
- Create: `internal/mcp/jsonrpc.go`
- Test: `internal/mcp/jsonrpc_test.go`

**Step 1: Write failing test for JSON-RPC request parsing**

```go
// internal/mcp/jsonrpc_test.go
package mcp

import (
	"encoding/json"
	"testing"
)

func TestParseRequest_ValidRequest(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value"}}`)

	req, err := parseRequest(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", req.JSONRPC)
	}
	if req.Method != "test" {
		t.Errorf("expected method test, got %s", req.Method)
	}
}

func TestParseRequest_MalformedJSON(t *testing.T) {
	data := []byte(`{"jsonrpc":"2.0","id":1`)

	_, err := parseRequest(data)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParseRequest_MissingJSONRPC(t *testing.T) {
	data := []byte(`{"id":1,"method":"test"}`)

	_, err := parseRequest(data)
	if err == nil {
		t.Fatal("expected error for missing jsonrpc field")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestParseRequest`
Expected: FAIL - undefined: parseRequest

**Step 3: Write minimal JSON-RPC types and parseRequest**

```go
// internal/mcp/jsonrpc.go
package mcp

import (
	"encoding/json"
	"fmt"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 success response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result"`
}

// JSONRPCErrorResponse represents a JSON-RPC 2.0 error response
type JSONRPCErrorResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      interface{}  `json:"id,omitempty"`
	Error   JSONRPCError `json:"error"`
}

// JSONRPCError represents a JSON-RPC error object
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID)
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// parseRequest parses a JSON-RPC request from bytes
func parseRequest(data []byte) (*JSONRPCRequest, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if req.JSONRPC != "2.0" {
		return nil, fmt.Errorf("invalid jsonrpc version: %s", req.JSONRPC)
	}

	if req.Method == "" {
		return nil, fmt.Errorf("missing required field: method")
	}

	return &req, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestParseRequest`
Expected: PASS

**Step 5: Add tests for response creation helpers**

```go
// internal/mcp/jsonrpc_test.go
func TestCreateResponse_Success(t *testing.T) {
	result := map[string]string{"status": "ok"}

	data, err := createResponse(1, result)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	if resp.ID != 1 {
		t.Errorf("expected id 1, got %v", resp.ID)
	}
}

func TestCreateErrorResponse_ProtocolError(t *testing.T) {
	data, err := createErrorResponse(1, -32601, "Method not found", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var resp JSONRPCErrorResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "Method not found" {
		t.Errorf("expected message 'Method not found', got %s", resp.Error.Message)
	}
}
```

**Step 6: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestCreate`
Expected: FAIL - undefined: createResponse, createErrorResponse

**Step 7: Implement response creation helpers**

```go
// internal/mcp/jsonrpc.go
// createResponse creates a JSON-RPC success response
func createResponse(id interface{}, result interface{}) ([]byte, error) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return json.Marshal(resp)
}

// createErrorResponse creates a JSON-RPC error response
func createErrorResponse(id interface{}, code int, message string, data interface{}) ([]byte, error) {
	resp := JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	return json.Marshal(resp)
}
```

**Step 8: Run tests to verify they pass**

Run: `go test ./internal/mcp -v`
Expected: PASS (all jsonrpc tests)

**Step 9: Commit**

```bash
git add internal/mcp/jsonrpc.go internal/mcp/jsonrpc_test.go
git commit -m "feat(mcp): add JSON-RPC message types and parsing

- Add JSONRPCRequest, JSONRPCResponse, JSONRPCErrorResponse types
- Add parseRequest for request validation
- Add createResponse and createErrorResponse helpers
- Test coverage for parsing and creation

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Error Code Constants

**Files:**
- Create: `internal/mcp/errors.go`
- Test: `internal/mcp/errors_test.go`

**Step 1: Write failing test for error constants**

```go
// internal/mcp/errors_test.go
package mcp

import "testing"

func TestErrorCodes_Defined(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{"ParseError", ErrCodeParseError},
		{"InvalidRequest", ErrCodeInvalidRequest},
		{"MethodNotFound", ErrCodeMethodNotFound},
		{"InvalidParams", ErrCodeInvalidParams},
		{"InternalError", ErrCodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code == 0 {
				t.Errorf("expected non-zero error code for %s", tt.name)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestErrorCodes`
Expected: FAIL - undefined: ErrCodeParseError, etc.

**Step 3: Implement error constants**

```go
// internal/mcp/errors.go
package mcp

// JSON-RPC 2.0 error codes
const (
	ErrCodeParseError     = -32700 // Invalid JSON
	ErrCodeInvalidRequest = -32600 // Missing required fields
	ErrCodeMethodNotFound = -32601 // Unknown method
	ErrCodeInvalidParams  = -32602 // Wrong parameter types, unknown tool
	ErrCodeInternalError  = -32603 // Server panic, unexpected failures
)

// Error messages
const (
	ErrMsgParseError     = "Parse error"
	ErrMsgInvalidRequest = "Invalid Request"
	ErrMsgMethodNotFound = "Method not found"
	ErrMsgInvalidParams  = "Invalid params"
	ErrMsgInternalError  = "Internal error"
)
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestErrorCodes`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/errors.go internal/mcp/errors_test.go
git commit -m "feat(mcp): add JSON-RPC error code constants

- Define standard error codes (-32700 to -32603)
- Add error message constants
- Test error code definitions

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Server State Management

**Files:**
- Modify: `internal/mcp/server.go`
- Test: `internal/mcp/server_test.go`

**Step 1: Write failing test for state management**

```go
// internal/mcp/server_test.go
package mcp

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestServer_InitialState(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	if srv.getState() != stateNotInitialized {
		t.Errorf("expected initial state NotInitialized, got %v", srv.getState())
	}
}

func TestServer_StateTransitions(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	// Transition to initializing
	srv.setState(stateInitializing)
	if srv.getState() != stateInitializing {
		t.Errorf("expected state Initializing, got %v", srv.getState())
	}

	// Transition to initialized
	srv.setState(stateInitialized)
	if srv.getState() != stateInitialized {
		t.Errorf("expected state Initialized, got %v", srv.getState())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestServer_`
Expected: FAIL - undefined: stateNotInitialized, getState, setState

**Step 3: Add state management to Server**

```go
// internal/mcp/server.go
package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

// serverState represents the server lifecycle state
type serverState int

const (
	stateNotInitialized serverState = iota
	stateInitializing
	stateInitialized
)

// Server implements the Model Context Protocol for TMKB
type Server struct {
	index              *knowledge.Index
	state              serverState
	protocolVersion    string
	clientCapabilities map[string]interface{}
	mu                 sync.RWMutex
}

// NewServer creates a new MCP server
func NewServer(index *knowledge.Index) *Server {
	return &Server{
		index: index,
		state: stateNotInitialized,
	}
}

// setState sets the server state (thread-safe)
func (s *Server) setState(state serverState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

// getState gets the server state (thread-safe)
func (s *Server) getState() serverState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Rest of existing methods...
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestServer_`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/server_test.go
git commit -m "feat(mcp): add server state management

- Add serverState enum (NotInitialized, Initializing, Initialized)
- Add setState and getState methods with mutex protection
- Update Server struct with state fields
- Test state transitions

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Input Validation Functions

**Files:**
- Create: `internal/mcp/validation.go`
- Test: `internal/mcp/validation_test.go`

**Step 1: Write failing tests for validation**

```go
// internal/mcp/validation_test.go
package mcp

import (
	"testing"
)

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid", "tmkb_query", false},
		{"Invalid", "other_tool", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContext(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"Valid", "background job processing", false, ""},
		{"Empty", "", true, "context must be non-empty"},
		{"Whitespace", "   ", true, "context must be non-empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContext(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContext(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("validateContext(%q) error message = %q, want %q", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Python", "python", false},
		{"Empty (optional)", "", false},
		{"Invalid Java", "java", true},
		{"Invalid Go", "go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLanguage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLanguage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFramework(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Flask", "flask", false},
		{"Valid Any", "any", false},
		{"Empty (optional)", "", false},
		{"Invalid Django", "django", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFramework(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFramework(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVerbosity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Agent", "agent", false},
		{"Valid Human", "human", false},
		{"Empty (optional)", "", false},
		{"Invalid Verbose", "verbose", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVerbosity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVerbosity(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNoUnknownParams(t *testing.T) {
	allowed := []string{"context", "language", "framework", "verbosity"}

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{"All allowed", map[string]interface{}{"context": "test", "language": "python"}, false},
		{"Unknown param", map[string]interface{}{"context": "test", "timeout": 30}, true},
		{"Empty", map[string]interface{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoUnknownParams(tt.args, allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNoUnknownParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestValidate`
Expected: FAIL - undefined: validateToolName, etc.

**Step 3: Implement validation functions**

```go
// internal/mcp/validation.go
package mcp

import (
	"fmt"
	"strings"
)

// validateToolName validates the tool name is "tmkb_query"
func validateToolName(name string) error {
	if name != "tmkb_query" {
		return fmt.Errorf("unknown tool: %s", name)
	}
	return nil
}

// validateContext validates the context parameter
func validateContext(context string) error {
	if strings.TrimSpace(context) == "" {
		return fmt.Errorf("context must be non-empty")
	}
	return nil
}

// validateLanguage validates the language parameter
func validateLanguage(language string) error {
	if language == "" {
		return nil // Optional field
	}

	validLanguages := []string{"python"}
	for _, valid := range validLanguages {
		if language == valid {
			return nil
		}
	}

	return fmt.Errorf("Invalid language '%s'. Supported languages: python", language)
}

// validateFramework validates the framework parameter
func validateFramework(framework string) error {
	if framework == "" {
		return nil // Optional field
	}

	validFrameworks := []string{"flask", "any"}
	for _, valid := range validFrameworks {
		if framework == valid {
			return nil
		}
	}

	return fmt.Errorf("Invalid framework '%s'. Supported frameworks: flask, any", framework)
}

// validateVerbosity validates the verbosity parameter
func validateVerbosity(verbosity string) error {
	if verbosity == "" {
		return nil // Optional field
	}

	validVerbosity := []string{"agent", "human"}
	for _, valid := range validVerbosity {
		if verbosity == valid {
			return nil
		}
	}

	return fmt.Errorf("Invalid verbosity '%s'. Supported values: agent, human", verbosity)
}

// validateNoUnknownParams checks for unknown parameters
func validateNoUnknownParams(args map[string]interface{}, allowed []string) error {
	allowedMap := make(map[string]bool)
	for _, key := range allowed {
		allowedMap[key] = true
	}

	for key := range args {
		if !allowedMap[key] {
			return fmt.Errorf("Unknown parameter '%s'. Supported parameters: %s", key, strings.Join(allowed, ", "))
		}
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestValidate`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/validation.go internal/mcp/validation_test.go
git commit -m "feat(mcp): add input validation functions

- Add validateToolName, validateContext validators
- Add validateLanguage, validateFramework, validateVerbosity
- Add validateNoUnknownParams for unknown parameter detection
- Comprehensive test coverage for all validators

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Initialize Handler

**Files:**
- Create: `internal/mcp/initialize.go`
- Test: `internal/mcp/initialize_test.go`

**Step 1: Write failing test for initialize handler**

```go
// internal/mcp/initialize_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestHandleInitialize`
Expected: FAIL - undefined: handleInitialize

**Step 3: Implement handleInitialize**

```go
// internal/mcp/initialize.go
package mcp

import (
	"encoding/json"
	"fmt"
)

// initializeParams represents the initialize request parameters
type initializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      map[string]interface{} `json:"clientInfo,omitempty"`
}

// initializeResult represents the initialize response result
type initializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      map[string]interface{} `json:"serverInfo"`
}

// handleInitialize handles the initialize request
func handleInitialize(s *Server, params json.RawMessage) (interface{}, error) {
	// Check if already initialized
	state := s.getState()
	if state != stateNotInitialized {
		return nil, fmt.Errorf("already initialized")
	}

	// Parse parameters
	var p initializeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid initialize params: %w", err)
	}

	// Version negotiation: support 2025-11-25 only
	protocolVersion := "2025-11-25"
	if p.ProtocolVersion != protocolVersion {
		// Client requested unsupported version, respond with our version
		// Client may disconnect if incompatible
		protocolVersion = "2025-11-25"
	}

	// Store protocol version and client capabilities
	s.mu.Lock()
	s.protocolVersion = protocolVersion
	s.clientCapabilities = p.Capabilities
	s.mu.Unlock()

	// Transition to initializing state
	s.setState(stateInitializing)

	// Build response
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		ServerInfo: map[string]interface{}{
			"name":        "tmkb",
			"version":     "0.1.0",
			"description": "Threat Model Knowledge Base - Query authorization security threats",
		},
	}

	return result, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestHandleInitialize`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/initialize.go internal/mcp/initialize_test.go
git commit -m "feat(mcp): implement initialize handler

- Add handleInitialize with version negotiation
- Return server capabilities (tools with listChanged: false)
- Transition server to initializing state
- Test success and duplicate init cases

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Tools Handler (List and Call)

**Files:**
- Create: `internal/mcp/tools.go`
- Test: `internal/mcp/tools_test.go`

**Step 1: Write failing test for tools/list**

```go
// internal/mcp/tools_test.go
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestHandleToolsList`
Expected: FAIL - undefined: handleToolsList

**Step 3: Write failing test for tools/call**

```go
// internal/mcp/tools_test.go (add to existing file)
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
```

**Step 4: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestHandleToolsCall`
Expected: FAIL - undefined: handleToolsCall

**Step 5: Implement handleToolsList and handleToolsCall**

```go
// internal/mcp/tools.go
package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

// toolsListResult represents the tools/list response
type toolsListResult struct {
	Tools []interface{} `json:"tools"`
}

// toolsCallParams represents the tools/call request parameters
type toolsCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// toolCallResult represents the tools/call response
type toolCallResult struct {
	Content []interface{} `json:"content"`
	IsError bool          `json:"isError"`
}

// handleToolsList handles the tools/list request
func handleToolsList(s *Server, params json.RawMessage) (interface{}, error) {
	// Check if initialized
	if s.getState() != stateInitialized {
		return nil, fmt.Errorf("server not initialized")
	}

	// Return tool definition
	tool := s.ToolDefinition()

	// Update tool schema with strict validation
	inputSchema := tool["inputSchema"].(map[string]interface{})
	properties := inputSchema["properties"].(map[string]interface{})

	// Add minLength to context
	context := properties["context"].(map[string]interface{})
	context["minLength"] = 1

	// Add additionalProperties: false
	inputSchema["additionalProperties"] = false

	result := toolsListResult{
		Tools: []interface{}{tool},
	}

	return result, nil
}

// handleToolsCall handles the tools/call request
func handleToolsCall(s *Server, params json.RawMessage) (interface{}, error) {
	// Check if initialized
	if s.getState() != stateInitialized {
		return nil, fmt.Errorf("server not initialized")
	}

	// Parse parameters
	var p toolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}

	// Validate tool name (protocol error)
	if err := validateToolName(p.Name); err != nil {
		return nil, err // Protocol error
	}

	// Extract and validate arguments
	context, _ := p.Arguments["context"].(string)
	language, _ := p.Arguments["language"].(string)
	framework, _ := p.Arguments["framework"].(string)
	verbosity, _ := p.Arguments["verbosity"].(string)

	// Validate context (tool execution error)
	if err := validateContext(context); err != nil {
		return createToolExecutionErrorResult(err.Error()), nil
	}

	// Validate language (tool execution error)
	if err := validateLanguage(language); err != nil {
		return createToolExecutionErrorResult(err.Error()), nil
	}

	// Validate framework (tool execution error)
	if err := validateFramework(framework); err != nil {
		return createToolExecutionErrorResult(err.Error()), nil
	}

	// Validate verbosity (tool execution error)
	if err := validateVerbosity(verbosity); err != nil {
		return createToolExecutionErrorResult(err.Error()), nil
	}

	// Check for unknown parameters (tool execution error)
	allowed := []string{"context", "language", "framework", "verbosity"}
	if err := validateNoUnknownParams(p.Arguments, allowed); err != nil {
		return createToolExecutionErrorResult(err.Error()), nil
	}

	// Execute query using existing HandleRequest
	queryResult, err := s.HandleRequest(p.Arguments)
	if err != nil {
		return createToolExecutionErrorResult(fmt.Sprintf("Query failed: %v", err)), nil
	}

	// Wrap result in MCP tool call format
	result := toolCallResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": queryResult,
			},
		},
		IsError: false,
	}

	return result, nil
}

// createToolExecutionErrorResult creates a tool execution error result
func createToolExecutionErrorResult(message string) interface{} {
	return toolCallResult{
		Content: []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": message,
			},
		},
		IsError: true,
	}
}
```

**Step 6: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestHandleTools`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/mcp/tools.go internal/mcp/tools_test.go
git commit -m "feat(mcp): implement tools/list and tools/call handlers

- Add handleToolsList with enhanced schema validation
- Add handleToolsCall with strict input validation
- Distinguish protocol errors from tool execution errors
- Test success cases and all validation errors

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 7: Message Routing and Handler Registry

**Files:**
- Create: `internal/mcp/handlers.go`
- Test: `internal/mcp/handlers_test.go`

**Step 1: Write failing test for message routing**

```go
// internal/mcp/handlers_test.go
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

	if resp.ID != 1 {
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestHandleMessage`
Expected: FAIL - undefined: handleMessage

**Step 3: Implement handler registry and routing**

```go
// internal/mcp/handlers.go
package mcp

import (
	"encoding/json"
	"fmt"
)

// Handler is a function that handles an MCP request
type Handler func(*Server, json.RawMessage) (interface{}, error)

// handlers maps method names to handler functions
var handlers = map[string]Handler{
	"initialize":  handleInitialize,
	"tools/list":  handleToolsList,
	"tools/call":  handleToolsCall,
}

// handleMessage processes a single JSON-RPC message
func (s *Server) handleMessage(msg []byte) ([]byte, error) {
	// Try to parse as request first
	var req JSONRPCRequest
	if err := json.Unmarshal(msg, &req); err == nil && req.ID != nil {
		return s.handleRequest(&req)
	}

	// Try to parse as notification
	var notif JSONRPCNotification
	if err := json.Unmarshal(msg, &notif); err == nil && notif.Method != "" {
		return s.handleNotification(&notif)
	}

	// Invalid message
	return createErrorResponse(nil, ErrCodeInvalidRequest, ErrMsgInvalidRequest, nil)
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(req *JSONRPCRequest) ([]byte, error) {
	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		return createErrorResponse(req.ID, ErrCodeInvalidRequest, "Invalid jsonrpc version", nil)
	}

	// Look up handler
	handler, ok := handlers[req.Method]
	if !ok {
		return createErrorResponse(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}

	// Call handler
	result, err := handler(s, req.Params)
	if err != nil {
		// Handler returned an error - this is a protocol error
		return createErrorResponse(req.ID, ErrCodeInvalidParams, err.Error(), nil)
	}

	// Success response
	return createResponse(req.ID, result)
}

// handleNotification processes a JSON-RPC notification
func (s *Server) handleNotification(notif *JSONRPCNotification) ([]byte, error) {
	// Handle initialized notification
	if notif.Method == "notifications/initialized" {
		if s.getState() == stateInitializing {
			s.setState(stateInitialized)
		}
		return []byte{}, nil // No response for notifications
	}

	// Unknown notification - ignore per JSON-RPC spec
	return []byte{}, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestHandleMessage`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/handlers.go internal/mcp/handlers_test.go
git commit -m "feat(mcp): implement message routing and handler registry

- Add handler registry mapping methods to handlers
- Add handleMessage for request/notification dispatch
- Handle initialized notification state transition
- Test message routing and method not found

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 8: stdio Transport Implementation

**Files:**
- Modify: `internal/mcp/server.go` (ServeStdio method)
- Test: `internal/mcp/stdio_test.go`

**Step 1: Write failing test for stdio transport**

```go
// internal/mcp/stdio_test.go
package mcp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestServeStdio_InitializeFlow(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
`

	var output bytes.Buffer
	err := srv.ServeStdio(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 response line, got %d", len(lines))
	}

	// First line should be initialize response
	if !strings.Contains(lines[0], `"protocolVersion":"2025-11-25"`) {
		t.Error("expected initialize response")
	}

	if srv.getState() != stateInitialized {
		t.Errorf("expected state Initialized, got %v", srv.getState())
	}
}

func TestServeStdio_MalformedJSON(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	input := `{"jsonrpc":"2.0","id":1,"method":`

	var output bytes.Buffer
	err := srv.ServeStdio(strings.NewReader(input), &output)

	// Should handle malformed JSON gracefully
	if err != nil && !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServeStdio_EmptyInput(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	input := ""

	var output bytes.Buffer
	err := srv.ServeStdio(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("expected no error for empty input, got %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcp -v -run TestServeStdio`
Expected: FAIL - ServeStdio doesn't implement protocol

**Step 3: Implement ServeStdio**

```go
// internal/mcp/server.go (replace existing ServeStdio)
import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

// ServeStdio runs the MCP server over stdin/stdout
func (s *Server) ServeStdio(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	writer := bufio.NewWriter(w)

	// Set max buffer size to 10MB
	const maxBufferSize = 10 * 1024 * 1024
	buf := make([]byte, maxBufferSize)
	scanner.Buffer(buf, maxBufferSize)

	for scanner.Scan() {
		msg := scanner.Bytes()

		// Handle message
		resp, err := s.handleMessage(msg)
		if err != nil {
			log.Printf("[ERROR] Failed to handle message: %v", err)
			continue
		}

		// Write response if non-empty (notifications have no response)
		if len(resp) > 0 {
			if _, err := writer.Write(resp); err != nil {
				log.Printf("[ERROR] Failed to write response: %v", err)
				return fmt.Errorf("failed to write response: %w", err)
			}
			if err := writer.WriteByte('\n'); err != nil {
				log.Printf("[ERROR] Failed to write newline: %v", err)
				return fmt.Errorf("failed to write newline: %w", err)
			}
			if err := writer.Flush(); err != nil {
				log.Printf("[ERROR] Failed to flush writer: %v", err)
				return fmt.Errorf("failed to flush writer: %w", err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// EOF is normal shutdown
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestServeStdio`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/mcp/server.go internal/mcp/stdio_test.go
git commit -m "feat(mcp): implement stdio transport with buffering

- Replace ServeStdio placeholder with full implementation
- Read newline-delimited JSON from stdin
- Write responses to stdout with buffering
- Handle EOF gracefully for shutdown
- Test initialize flow and error cases

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Integration Tests

**Files:**
- Create: `internal/mcp/integration_test.go`

**Step 1: Write integration tests**

```go
// internal/mcp/integration_test.go
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
	if initResp.ID != 1 {
		t.Errorf("expected id 1, got %v", initResp.ID)
	}

	// Verify tools/list response
	var listResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to parse list response: %v", err)
	}
	if listResp.ID != 2 {
		t.Errorf("expected id 2, got %v", listResp.ID)
	}

	// Verify tools/call response
	var callResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[2]), &callResp); err != nil {
		t.Fatalf("failed to parse call response: %v", err)
	}
	if callResp.ID != 3 {
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
	if listResp.ID != 2 {
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
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/mcp -v -run TestIntegration`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/mcp/integration_test.go
git commit -m "test(mcp): add end-to-end integration tests

- Test full session flow (initialize through tool call)
- Test error recovery (invalid followed by valid request)
- Test validation errors return actionable messages
- Use real patterns for realistic testing

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Task 10: Test Coverage Validation and Documentation

**Files:**
- Create: `docs/validation/2026-02-06-mcp-protocol-validation.md`

**Step 1: Run all tests and check coverage**

Run: `go test ./internal/mcp/... -v -cover`
Expected: All tests pass, >80% coverage

**Step 2: Run integration tests with real patterns**

Run: `go test ./internal/mcp -v -run TestIntegration`
Expected: PASS (or SKIP if patterns not available)

**Step 3: Create validation report**

```bash
# Generate coverage report
go test ./internal/mcp/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Count tests
go test ./internal/mcp/... -v | grep -c "^=== RUN"
```

**Step 4: Write validation report**

Create: `docs/validation/2026-02-06-mcp-protocol-validation.md`

```markdown
# MCP Protocol Implementation Validation

**Date**: 2026-02-06
**Issue**: #6
**Status**: Complete

## Summary

Implemented complete JSON-RPC 2.0 protocol support for MCP server with stdio transport. All tests passing, coverage exceeds target.

## Test Results

**Total Tests**: <N> passing
- JSON-RPC parsing: <N> tests
- Error handling: <N> tests
- Initialize handler: <N> tests
- Tools handlers: <N> tests
- Message routing: <N> tests
- stdio transport: <N> tests
- Integration: <N> tests

**Coverage**: <X>% (target >80%)

## Functionality Verification

✅ Full JSON-RPC 2.0 protocol support
✅ Initialize/initialized lifecycle working
✅ Tools/list returns correct definition
✅ Tools/call validates input strictly
✅ All error cases handled correctly
✅ Graceful shutdown on EOF

## Manual Testing

Tested with sample inputs:

\`\`\`bash
# Initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{}}}' | go run cmd/tmkb/main.go serve

# Expected: Initialize response with server capabilities
\`\`\`

## Files Created

- `internal/mcp/jsonrpc.go` - JSON-RPC message types
- `internal/mcp/errors.go` - Error code constants
- `internal/mcp/validation.go` - Input validation
- `internal/mcp/initialize.go` - Initialize handler
- `internal/mcp/tools.go` - Tools handlers
- `internal/mcp/handlers.go` - Message routing
- Test files: 7 files with comprehensive coverage

## Success Criteria

✅ >80% test coverage
✅ All handlers have unit tests
✅ End-to-end integration tests pass
✅ Error paths tested comprehensively
✅ Clear error messages for LLM self-correction
✅ Following Go best practices
```

**Step 5: Commit validation report**

```bash
git add docs/validation/2026-02-06-mcp-protocol-validation.md
git commit -m "docs(mcp): add protocol validation report

Validates Issue #6 completion:
- All tests passing (>80% coverage)
- Full protocol implementation verified
- Integration tests with real patterns pass

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

## Success Criteria

### Functionality
- ✅ Full JSON-RPC 2.0 protocol support
- ✅ Initialize/initialized lifecycle working
- ✅ Tools/list returns correct definition
- ✅ Tools/call validates input strictly
- ✅ All error cases handled correctly
- ✅ Graceful shutdown on EOF

### Testing
- ✅ >80% test coverage
- ✅ All handlers have unit tests
- ✅ End-to-end integration tests pass
- ✅ Error paths tested comprehensively

### Quality
- ✅ Clear error messages for LLM self-correction
- ✅ No data races (concurrent requests handled safely)
- ✅ Clean separation of concerns (handlers, validation, routing)
- ✅ Following Go best practices

## Notes

- Each task follows TDD: test first, watch fail, implement, watch pass, commit
- Tasks are bite-sized (2-5 minutes per step)
- Complete code provided in plan (no "add validation" handwaving)
- Tests verify both success and error paths
- Integration tests use real patterns for realistic validation
