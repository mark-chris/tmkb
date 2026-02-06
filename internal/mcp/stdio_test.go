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
