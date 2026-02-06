package mcp

import (
	"testing"

	"github.com/mark-chris/tmkb/internal/knowledge"
)

func TestServer_InitialState(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	if srv.getState() != stateNotInitialized {
		t.Errorf("expected initial state NotInitialized, got %v", srv.getState())
	}
}

func TestServer_StateTransitions(t *testing.T) {
	idx := knowledge.NewIndex()
	srv := NewServer(idx)

	// Transition to initializing
	srv.setState(stateInitializing)
	if srv.getState() != stateInitializing {
		t.Errorf("expected state Initializing, got %v", srv.getState())
	}

	// Transition to initialized
	srv.setState(stateInitialized)
	if srv.getState() != stateInitialized {
		t.Errorf("expected state Initialized, got %v", srv.getState())
	}
}
