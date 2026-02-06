package mcp

import (
	"fmt"
	"strings"
)

// validateToolName validates the tool name is "tmkb_query"
func validateToolName(name string) error {
	if name != "tmkb_query" {
		return fmt.Errorf("unknown tool: %s", name)
	}
	return nil
}

// validateContext validates the context parameter
func validateContext(context string) error {
	if strings.TrimSpace(context) == "" {
		return fmt.Errorf("context must be non-empty")
	}
	return nil
}

// validateLanguage validates the language parameter
func validateLanguage(language string) error {
	if language == "" {
		return nil // Optional field
	}

	validLanguages := []string{"python"}
	for _, valid := range validLanguages {
		if language == valid {
			return nil
		}
	}

	return fmt.Errorf("Invalid language '%s'. Supported languages: python", language)
}

// validateFramework validates the framework parameter
func validateFramework(framework string) error {
	if framework == "" {
		return nil // Optional field
	}

	validFrameworks := []string{"flask", "any"}
	for _, valid := range validFrameworks {
		if framework == valid {
			return nil
		}
	}

	return fmt.Errorf("Invalid framework '%s'. Supported frameworks: flask, any", framework)
}

// validateVerbosity validates the verbosity parameter
func validateVerbosity(verbosity string) error {
	if verbosity == "" {
		return nil // Optional field
	}

	validVerbosity := []string{"agent", "human"}
	for _, valid := range validVerbosity {
		if verbosity == valid {
			return nil
		}
	}

	return fmt.Errorf("Invalid verbosity '%s'. Supported values: agent, human", verbosity)
}

// validateNoUnknownParams checks for unknown parameters
func validateNoUnknownParams(args map[string]interface{}, allowed []string) error {
	allowedMap := make(map[string]bool)
	for _, key := range allowed {
		allowedMap[key] = true
	}

	for key := range args {
		if !allowedMap[key] {
			return fmt.Errorf("Unknown parameter '%s'. Supported parameters: %s", key, strings.Join(allowed, ", "))
		}
	}

	return nil
}
