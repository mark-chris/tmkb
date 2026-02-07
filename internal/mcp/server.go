package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
