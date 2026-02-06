package mcp

import (
	"encoding/json"
	"fmt"
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

	result := map[string]interface{}{
		"tools": []interface{}{tool},
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
	result := map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": queryResult,
			},
		},
		"isError": false,
	}

	return result, nil
}

// createToolExecutionErrorResult creates a tool execution error result
func createToolExecutionErrorResult(message string) interface{} {
	return map[string]interface{}{
		"content": []interface{}{
			map[string]interface{}{
				"type": "text",
				"text": message,
			},
		},
		"isError": true,
	}
}
