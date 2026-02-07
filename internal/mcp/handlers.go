package mcp

import (
	"encoding/json"
	"fmt"
)

// Handler is a function that handles an MCP request
type Handler func(*Server, json.RawMessage) (interface{}, error)

// handlers maps method names to handler functions
var handlers = map[string]Handler{
	"initialize": handleInitialize,
	"tools/list": handleToolsList,
	"tools/call": handleToolsCall,
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
	errResp := createErrorResponse(ErrCodeInvalidRequest, ErrMsgInvalidRequest, nil, nil)
	return json.Marshal(errResp)
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(req *JSONRPCRequest) ([]byte, error) {
	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		errResp := createErrorResponse(ErrCodeInvalidRequest, "Invalid jsonrpc version", nil, req.ID)
		return json.Marshal(errResp)
	}

	// Look up handler
	handler, ok := handlers[req.Method]
	if !ok {
		errResp := createErrorResponse(ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil, req.ID)
		return json.Marshal(errResp)
	}

	// Call handler
	result, err := handler(s, req.Params)
	if err != nil {
		// Handler returned an error - this is a protocol error
		errResp := createErrorResponse(ErrCodeInvalidParams, err.Error(), nil, req.ID)
		return json.Marshal(errResp)
	}

	// Success response
	resp := createResponse(result, req.ID)
	return json.Marshal(resp)
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
