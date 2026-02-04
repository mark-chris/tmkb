package mcp

// Handler processes MCP protocol messages
type Handler struct {
	server *Server
}

// NewHandler creates a new MCP handler
func NewHandler(server *Server) *Handler {
	return &Handler{server: server}
}

// HandleToolCall processes a tool call request
func (h *Handler) HandleToolCall(toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "tmkb_query":
		return h.server.HandleRequest(args)
	default:
		return nil, nil
	}
}

// GetTools returns the list of available tools
func (h *Handler) GetTools() []map[string]interface{} {
	return []map[string]interface{}{
		h.server.ToolDefinition(),
	}
}
