# MCP Protocol Implementation Design

**Date**: 2026-02-06
**Issue**: #6
**Status**: Approved

## Overview

Implement complete JSON-RPC 2.0 protocol support for the MCP (Model Context Protocol) server, enabling TMKB to integrate with Claude Desktop and other MCP clients over stdio transport.

## Goals

1. Implement full JSON-RPC 2.0 message handling (requests, responses, notifications, errors)
2. Implement MCP lifecycle (initialize, initialized notification, operational state)
3. Implement tool discovery (`tools/list`) and invocation (`tools/call`)
4. Strict input validation with actionable error messages for LLM self-correction
5. Comprehensive test coverage (>80%)

## Current State Assessment

### Existing Implementation

**File: `internal/mcp/server.go`**
- ✅ Basic `Server` struct with knowledge index
- ✅ `ToolDefinition()` method with input schema
- ✅ `HandleRequest()` method that executes queries
- ❌ No JSON-RPC protocol handling
- ❌ No lifecycle management (initialize/initialized)
- ❌ No message routing or handler registry
- ❌ Incomplete `ServeStdio()` (placeholder only)

**File: `cmd/tmkb/serve.go`**
- ✅ CLI command that calls `ServeStdio()`
- ✅ Pattern initialization
- ❌ No actual protocol implementation

### What's Missing

- ❌ JSON-RPC message parsing and validation
- ❌ Request/response envelope handling
- ❌ Handler registry pattern
- ❌ Initialize/initialized lifecycle
- ❌ Tools/list endpoint
- ❌ Tools/call with strict validation
- ❌ Protocol error handling
- ❌ Tool execution error handling
- ❌ State machine (uninitialized → initialized → operational)
- ❌ MCP protocol tests

## Design Decisions

### Transport: stdio Only

**Decision**: Implement stdio transport only, not HTTP.

**Rationale**:
- stdio is the standard for Claude Desktop integration
- Simpler implementation (no HTTP server, SSE, authentication)
- Matches CLI tool pattern
- HTTP can be added later if needed

**Alternative**: HTTP with SSE. Rejected because it adds complexity without immediate value for the primary use case (Claude Desktop).

### Architecture: Handler Registry Pattern

**Decision**: Use handler registry pattern for message routing.

**Rationale**:
- Clean separation of concerns (each handler is self-contained)
- Easy to unit test handlers independently
- Simple to add new methods later
- No over-engineering (avoids complex middleware chains)

**Alternative Considered**:
- Simple switch statement: Gets messy with many handlers
- Middleware chain: Over-engineered for 3-4 methods

**Implementation**:
```go
type Handler func(*Server, json.RawMessage) (interface{}, error)

handlers := map[string]Handler{
    "initialize":  handleInitialize,
    "tools/list":  handleToolsList,
    "tools/call":  handleToolsCall,
}
```

### Error Handling: Strict Validation

**Decision**: Strict validation with actionable error messages.

**Rationale**:
- Helps LLMs learn correct parameter space
- Clear distinction between protocol errors and tool execution errors
- Better security (reject invalid inputs explicitly)
- Guides LLM to self-correct

**Protocol Errors** (JSON-RPC error codes):
- Unknown tool
- Malformed requests
- Invalid JSON
- Method not found

**Tool Execution Errors** (isError: true):
- Invalid enum values (language, framework)
- Empty required fields (context)
- Unknown parameters

**Alternative**: Permissive fallback (silently ignore invalid values). Rejected because it's less transparent and doesn't help LLMs learn.

## Architecture

### Three-Layer Design

**1. Transport Layer** (`ServeStdio`)
- Reads newline-delimited JSON from stdin
- Writes JSON responses to stdout
- Buffers output to ensure atomic writes
- Handles EOF gracefully (shutdown)

**2. Message Routing Layer** (`handleMessage`)
- Parses JSON-RPC messages
- Validates JSON-RPC structure
- Routes to appropriate handler
- Wraps responses in JSON-RPC envelopes
- Handles protocol errors

**3. Handler Layer**
- `handleInitialize` - Capability negotiation
- `handleToolsList` - Returns tool definitions
- `handleToolsCall` - Validates and executes queries

### State Management

**State Machine:**
```
NotInitialized → Initializing → Initialized → Shutdown
```

**Rules:**
- Before `initialized` notification: Only accept `initialize` requests
- After `initialized`: Accept `tools/list` and `tools/call`
- Enforce state transitions strictly

**State Storage:**
```go
type Server struct {
    index             *knowledge.Index
    state             serverState
    protocolVersion   string
    clientCapabilities map[string]interface{}
    mu                sync.RWMutex  // Protect state
}

type serverState int

const (
    stateNotInitialized serverState = iota
    stateInitializing
    stateInitialized
)
```

## Protocol Implementation

### Message Types

**JSON-RPC Request:**
```go
type JSONRPCRequest struct {
    JSONRPC string          `json:"jsonrpc"`  // Must be "2.0"
    ID      interface{}     `json:"id"`       // string or number, not null
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}
```

**JSON-RPC Success Response:**
```go
type JSONRPCResponse struct {
    JSONRPC string      `json:"jsonrpc"`  // Must be "2.0"
    ID      interface{} `json:"id"`
    Result  interface{} `json:"result"`
}
```

**JSON-RPC Error Response:**
```go
type JSONRPCErrorResponse struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Error   JSONRPCError `json:"error"`
}

type JSONRPCError struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

**JSON-RPC Notification:**
```go
type JSONRPCNotification struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}
// No ID field - distinguishes from requests
```

### Error Codes

**Standard JSON-RPC Errors:**
| Code | Meaning | Usage |
|------|---------|-------|
| -32700 | Parse error | Invalid JSON |
| -32600 | Invalid Request | Missing required fields |
| -32601 | Method not found | Unknown method |
| -32602 | Invalid params | Wrong parameter types, unknown tool |
| -32603 | Internal error | Server panic, unexpected failures |

### Initialization Flow

**1. Client sends `initialize` request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "roots": {"listChanged": true},
      "sampling": {}
    },
    "clientInfo": {
      "name": "Claude Desktop",
      "version": "1.0.0"
    }
  }
}
```

**2. Server responds with capabilities:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "tools": {
        "listChanged": false
      }
    },
    "serverInfo": {
      "name": "tmkb",
      "version": "0.1.0",
      "description": "Threat Model Knowledge Base - Query authorization security threats"
    }
  }
}
```

**3. Client sends `initialized` notification:**
```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

**Protocol Version Negotiation:**
- Server supports: `2025-11-25` (latest MCP spec)
- If client requests supported version → respond with same version
- If client requests unsupported version → respond with server's latest version
- Client may disconnect if incompatible

**Capabilities:**
- Server declares `tools` capability with `listChanged: false` (static tool list)
- Server does not support: resources, prompts, logging, sampling, roots

### Tool Discovery

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "tmkb_query",
        "description": "Query the Threat Model Knowledge Base for authorization security threats relevant to your implementation. Returns concise, actionable security context optimized for code generation.",
        "inputSchema": {
          "type": "object",
          "properties": {
            "context": {
              "type": "string",
              "description": "What you're implementing (e.g., 'multi-tenant API endpoint', 'background job processing')",
              "minLength": 1
            },
            "language": {
              "type": "string",
              "enum": ["python"],
              "description": "Programming language (MVP: Python only)"
            },
            "framework": {
              "type": "string",
              "enum": ["flask", "any"],
              "description": "Framework context (MVP: Flask or any)"
            },
            "verbosity": {
              "type": "string",
              "enum": ["agent", "human"],
              "default": "agent",
              "description": "Output format: 'agent' for concise, 'human' for detailed"
            }
          },
          "required": ["context"],
          "additionalProperties": false
        }
      }
    ]
  }
}
```

**Tool Schema Enhancements:**
- Add `minLength: 1` to context (enforce non-empty)
- Add `additionalProperties: false` (reject unknown parameters)
- Keep strict enum values for validation

### Tool Invocation

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "tmkb_query",
    "arguments": {
      "context": "background job processing",
      "language": "python",
      "framework": "flask"
    }
  }
}
```

**Success Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"pattern_count\":3,\"patterns\":[...]}"
      }
    ],
    "isError": false
  }
}
```

**Tool Execution Error (Validation Failure):**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "Invalid language 'java'. Supported languages: python"
      }
    ],
    "isError": true
  }
}
```

**Protocol Error (Unknown Tool):**
```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "error": {
    "code": -32602,
    "message": "Unknown tool: invalid_tool_name"
  }
}
```

### Validation Rules

**Tool Name Validation:**
- Must be exactly "tmkb_query"
- Case-sensitive
- If wrong → Protocol error -32602

**Context Validation:**
- Required field
- Must be non-empty string (after trimming)
- If missing → Tool execution error "context parameter is required"
- If empty → Tool execution error "context must be non-empty"

**Language Validation:**
- Optional field
- If provided, must be in ["python"]
- If invalid → Tool execution error "Invalid language '<value>'. Supported languages: python"

**Framework Validation:**
- Optional field
- If provided, must be in ["flask", "any"]
- If invalid → Tool execution error "Invalid framework '<value>'. Supported frameworks: flask, any"

**Verbosity Validation:**
- Optional field (defaults to "agent")
- If provided, must be in ["agent", "human"]
- If invalid → Tool execution error "Invalid verbosity '<value>'. Supported values: agent, human"

**Unknown Parameters:**
- Reject with tool execution error "Unknown parameter '<name>'. Supported parameters: context, language, framework, verbosity"

## Error Handling

### Edge Cases

| Scenario | Handling |
|----------|----------|
| Multiple `initialize` requests | Error: "Already initialized" |
| Requests before `initialized` notification | Error: "Server not initialized. Send initialized notification first." |
| Empty stdin (EOF) | Graceful shutdown, exit cleanly |
| Malformed JSON | Parse error response (-32700) |
| Missing request ID | Error response without ID (per JSON-RPC spec) |
| Notifications (no ID) | Process but don't respond (e.g., `initialized`) |
| Very large messages | Buffer limit (10MB), reject with error |
| Unknown method | Method not found error (-32601) |
| Query engine error | Tool execution error with sanitized message |

### Logging Strategy

- Log all errors to stderr (separate from stdout protocol messages)
- Include request ID in error logs for debugging
- Don't expose internal errors to client (sanitize messages)
- Log level: INFO for normal operations, ERROR for failures

**Example Log Format:**
```
[ERROR] [req_id=3] Invalid language parameter: java
[INFO] [req_id=4] Query executed successfully: 3 patterns returned
```

## Testing Strategy

### Unit Tests

**1. Message Parsing Tests** (`jsonrpc_test.go`)
- Valid JSON-RPC requests/responses/notifications
- Malformed JSON → parse errors
- Missing required fields → invalid request errors
- Invalid message types

**2. Handler Tests**

`initialize_test.go`:
- TestHandleInitialize_Success
- TestHandleInitialize_VersionNegotiation
- TestHandleInitialize_DuplicateInit
- TestHandleInitialize_InvalidParams

`tools_list_test.go`:
- TestHandleToolsList_Success
- TestHandleToolsList_BeforeInit

`tools_call_test.go`:
- TestHandleToolsCall_Success
- TestHandleToolsCall_UnknownTool
- TestHandleToolsCall_MissingContext
- TestHandleToolsCall_EmptyContext
- TestHandleToolsCall_InvalidLanguage
- TestHandleToolsCall_InvalidFramework
- TestHandleToolsCall_InvalidVerbosity
- TestHandleToolsCall_UnknownParameter
- TestHandleToolsCall_QueryEngineIntegration

**3. Validation Tests** (`validation_test.go`)
- TestValidateToolName
- TestValidateContext
- TestValidateLanguage
- TestValidateFramework
- TestValidateVerbosity
- TestValidateUnknownParameters
- TestErrorMessageClarity

**4. State Machine Tests** (`lifecycle_test.go`)
- TestStateTransitions
- TestEnforceInitializationOrder
- TestRejectRequestsBeforeInit
- TestAcceptRequestsAfterInit
- TestHandleShutdown

### Integration Tests

**1. End-to-End Protocol Flow** (`integration_test.go`)
- TestFullSession_InitializeToShutdown
- TestMultipleToolCalls
- TestErrorRecovery (invalid request followed by valid)
- TestNotificationHandling

**2. stdio Transport Tests** (`stdio_test.go`)
- TestReadFromStdin
- TestWriteToStdout
- TestMessageFraming (newline-delimited JSON)
- TestBufferManagement
- TestEOFHandling

### Test Utilities

**File: `internal/mcp/testutil/testutil.go`**
```go
package testutil

// Request builders
func CreateInitializeRequest(id int, version string) []byte
func CreateToolsListRequest(id int) []byte
func CreateToolsCallRequest(id int, tool string, args map[string]interface{}) []byte
func CreateNotification(method string, params map[string]interface{}) []byte

// Response parsers
func ParseResponse(data []byte) (*Response, error)
func ParseError(data []byte) (*ErrorResponse, error)

// Assertions
func AssertJSONRPCError(t *testing.T, resp *Response, expectedCode int)
func AssertToolExecutionError(t *testing.T, resp *Response, expectedMessage string)
func AssertSuccess(t *testing.T, resp *Response)
func AssertStateEquals(t *testing.T, server *Server, expectedState serverState)
```

### Coverage Target

- **Unit tests**: >85% coverage for MCP package
- **Integration tests**: All critical paths covered
- **Total**: >80% overall coverage (matching CLI test coverage)

## Implementation Details

### File Structure

```
internal/mcp/
├── server.go              # Server struct, ServeStdio, state management
├── handlers.go            # Handler registry, message routing
├── initialize.go          # handleInitialize implementation
├── tools.go               # handleToolsList, handleToolsCall
├── validation.go          # Input validation functions
├── jsonrpc.go            # JSON-RPC message types and helpers
├── errors.go             # Error code constants and helpers
├── server_test.go        # Server and ServeStdio tests
├── handlers_test.go      # Message routing tests
├── initialize_test.go    # Initialize handler tests
├── tools_test.go         # Tools handler tests
├── validation_test.go    # Validation tests
├── lifecycle_test.go     # State machine tests
├── integration_test.go   # End-to-end tests
└── testutil/
    └── testutil.go       # Test utilities
```

### Key Functions

**Server Methods:**
```go
func (s *Server) ServeStdio(r io.Reader, w io.Writer) error
func (s *Server) handleMessage(msg []byte) ([]byte, error)
func (s *Server) route(method string, params json.RawMessage) (interface{}, error)
func (s *Server) setState(state serverState)
func (s *Server) getState() serverState
```

**Handlers:**
```go
func handleInitialize(s *Server, params json.RawMessage) (interface{}, error)
func handleToolsList(s *Server, params json.RawMessage) (interface{}, error)
func handleToolsCall(s *Server, params json.RawMessage) (interface{}, error)
```

**Validation:**
```go
func validateToolName(name string) error
func validateContext(context string) error
func validateLanguage(language string) error
func validateFramework(framework string) error
func validateVerbosity(verbosity string) error
func validateNoUnknownParams(args map[string]interface{}, allowed []string) error
```

**JSON-RPC Helpers:**
```go
func parseRequest(data []byte) (*JSONRPCRequest, error)
func parseNotification(data []byte) (*JSONRPCNotification, error)
func createResponse(id interface{}, result interface{}) ([]byte, error)
func createErrorResponse(id interface{}, code int, message string, data interface{}) ([]byte, error)
func createToolExecutionError(id interface{}, message string) ([]byte, error)
```

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

## Trade-offs

### stdio Only vs HTTP Support

**Chosen**: stdio only

**Pro**: Simpler, matches primary use case (Claude Desktop)
**Con**: Can't access remotely, no web UI

**Decision**: stdio meets MVP needs, HTTP can be added later if needed

### Strict Validation vs Permissive

**Chosen**: Strict validation with clear errors

**Pro**: Better security, helps LLMs learn, more transparent
**Con**: More code, requires comprehensive error messages

**Decision**: Security and LLM learning benefits outweigh complexity

### Handler Registry vs Switch Statement

**Chosen**: Handler registry pattern

**Pro**: Clean separation, easy testing, scales well
**Con**: Slightly more code than simple switch

**Decision**: Maintainability and testability benefits worth it

## Future Enhancements

1. **HTTP Transport** - Add SSE support for web-based clients
2. **Additional Tools** - Add tools for validation, schema introspection
3. **Resources** - Expose patterns as MCP resources
4. **Prompts** - Add prompt templates for common queries
5. **Logging** - Implement MCP logging capability for debugging
6. **Metrics** - Track tool usage, query performance

## References

- [MCP Specification (2025-11-25)](https://modelcontextprotocol.io/specification/2025-11-25)
- [MCP Basic Protocol](https://modelcontextprotocol.io/specification/2025-11-25/basic)
- [MCP Lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle)
- [MCP Tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- Issue #6: MCP Protocol Implementation
- Existing code: `internal/mcp/server.go`, `cmd/tmkb/serve.go`
