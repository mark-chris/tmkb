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

// ToolDefinition returns the MCP tool definition for tmkb_query
func (s *Server) ToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "tmkb_query",
		"description": "Query the Threat Model Knowledge Base for authorization security threats relevant to your implementation. Returns concise, actionable security context optimized for code generation.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"context": map[string]interface{}{
					"type":        "string",
					"description": "What you're implementing (e.g., 'multi-tenant API endpoint', 'background job processing', 'admin dashboard')",
				},
				"language": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"python"},
					"description": "Programming language (MVP: Python only)",
				},
				"framework": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"flask", "any"},
					"description": "Framework context (MVP: Flask only)",
				},
				"verbosity": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"agent", "human"},
					"default":     "agent",
					"description": "Output format: 'agent' for concise, 'human' for detailed",
				},
			},
			"required": []string{"context"},
		},
	}
}

// HandleRequest processes an MCP tool call
func (s *Server) HandleRequest(input map[string]interface{}) (string, error) {
	// Extract parameters
	context, _ := input["context"].(string)
	language, _ := input["language"].(string)
	framework, _ := input["framework"].(string)
	verbosity, _ := input["verbosity"].(string)

	if verbosity == "" {
		verbosity = "agent"
	}

	// Build query options
	opts := knowledge.QueryOptions{
		Context:   context,
		Language:  language,
		Framework: framework,
		Verbosity: verbosity,
		Limit:     3,
	}

	// Execute query
	result := knowledge.Query(s.index, opts)

	// Return JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(data), nil
}

// ServeStdio runs the MCP server over stdin/stdout
// This is a simplified implementation; full MCP requires more protocol handling
func (s *Server) ServeStdio(r io.Reader, w io.Writer) error {
	// TODO: Implement full MCP protocol
	// For now, this is a placeholder that shows the intended interface
	
	fmt.Fprintln(w, "MCP server ready")
	fmt.Fprintln(w, "Tool available: tmkb_query")
	
	return nil
}
