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

// supportedProtocolVersions lists the MCP protocol revisions this server can
// speak. The server echoes the client's requested version when it appears
// here; otherwise it falls back to defaultProtocolVersion.
var supportedProtocolVersions = map[string]bool{
	"2025-06-18": true,
	"2025-03-26": true,
	"2024-11-05": true,
}

// defaultProtocolVersion is returned when the client requests a version this
// server does not support. It must be a revision real MCP clients accept, so a
// standards-compliant client can still complete the handshake.
const defaultProtocolVersion = "2025-06-18"

// negotiateProtocolVersion implements MCP version negotiation: echo the
// client's requested version when supported, otherwise return our default.
func negotiateProtocolVersion(requested string) string {
	if supportedProtocolVersions[requested] {
		return requested
	}
	return defaultProtocolVersion
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

	// Negotiate the protocol version: echo the client's requested version when
	// supported, otherwise fall back to our default.
	protocolVersion := negotiateProtocolVersion(p.ProtocolVersion)

	// Store protocol version and client capabilities
	s.mu.Lock()
	s.protocolVersion = protocolVersion
	s.clientCapabilities = p.Capabilities
	s.mu.Unlock()

	// Transition to initializing state
	s.setState(stateInitializing)

	// Build response
	result := map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":        "tmkb",
			"version":     "0.1.0",
			"description": "Threat Model Knowledge Base - Query authorization security threats",
		},
	}

	return result, nil
}
