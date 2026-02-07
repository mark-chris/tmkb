package mcp

import "testing"

func TestErrorCodes_Defined(t *testing.T) {
	tests := []struct {
		name      string
		errorCode int
	}{
		{"ErrCodeParseError", ErrCodeParseError},
		{"ErrCodeInvalidRequest", ErrCodeInvalidRequest},
		{"ErrCodeMethodNotFound", ErrCodeMethodNotFound},
		{"ErrCodeInvalidParams", ErrCodeInvalidParams},
		{"ErrCodeInternalError", ErrCodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.errorCode == 0 {
				t.Errorf("%s = 0, want non-zero", tt.name)
			}
		})
	}
}
