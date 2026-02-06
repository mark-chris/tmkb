package mcp

import (
	"encoding/json"
	"errors"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	ID      interface{} `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCErrorResponse represents a JSON-RPC 2.0 error response
type JSONRPCErrorResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	Error   JSONRPCError `json:"error"`
	ID      interface{}  `json:"id"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (request without ID)
type JSONRPCNotification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// parseRequest parses and validates a JSON-RPC request
func parseRequest(data []byte) (*JSONRPCRequest, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}

	if req.JSONRPC != "2.0" {
		return nil, errors.New("invalid or missing jsonrpc version")
	}

	return &req, nil
}

// createResponse creates a JSON-RPC 2.0 success response
func createResponse(result interface{}, id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// createErrorResponse creates a JSON-RPC 2.0 error response
func createErrorResponse(code int, message string, data interface{}, id interface{}) JSONRPCErrorResponse {
	return JSONRPCErrorResponse{
		JSONRPC: "2.0",
		Error: JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
}
