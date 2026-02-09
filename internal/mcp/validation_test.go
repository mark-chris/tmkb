package mcp

import (
	"strings"
	"testing"
)

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid", "tmkb_query", false},
		{"Invalid", "other_tool", true},
		{"Empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContext(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{"Valid", "background job processing", false, ""},
		{"Empty", "", true, "context must be non-empty"},
		{"Whitespace", "   ", true, "context must be non-empty"},
		{"At max length", strings.Repeat("a", maxContextLength), false, ""},
		{"Exceeds max length", strings.Repeat("a", maxContextLength+1), true, "context exceeds maximum length of 10000 characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContext(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContext(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("validateContext(%q) error message = %q, want %q", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Python", "python", false},
		{"Empty (optional)", "", false},
		{"Invalid Java", "java", true},
		{"Invalid Go", "go", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLanguage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateLanguage(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFramework(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Flask", "flask", false},
		{"Valid Any", "any", false},
		{"Empty (optional)", "", false},
		{"Invalid Django", "django", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFramework(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFramework(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateVerbosity(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"Valid Agent", "agent", false},
		{"Valid Human", "human", false},
		{"Empty (optional)", "", false},
		{"Invalid Verbose", "verbose", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVerbosity(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVerbosity(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateNoUnknownParams(t *testing.T) {
	allowed := []string{"context", "language", "framework", "verbosity"}

	tests := []struct {
		name    string
		args    map[string]interface{}
		wantErr bool
	}{
		{"All allowed", map[string]interface{}{"context": "test", "language": "python"}, false},
		{"Unknown param", map[string]interface{}{"context": "test", "timeout": 30}, true},
		{"Empty", map[string]interface{}{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoUnknownParams(tt.args, allowed)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNoUnknownParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
