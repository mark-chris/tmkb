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
