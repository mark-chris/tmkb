package knowledge

import (
	tiktoken "github.com/pkoukk/tiktoken-go"
)

// TokenCounter provides token counting functionality
type TokenCounter struct {
	encoder *tiktoken.Tiktoken
}

// NewTokenCounter creates a new token counter with cl100k_base encoding
func NewTokenCounter() (*TokenCounter, error) {
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return &TokenCounter{encoder: nil}, err
	}
	return &TokenCounter{encoder: enc}, nil
}

// CountTokens counts the number of tokens in the given text
// Falls back to character/4 approximation if encoder is unavailable
func (tc *TokenCounter) CountTokens(text string) int {
	if tc.encoder == nil {
		// Fallback: approximate with character count
		return len(text) / 4
	}
	return len(tc.encoder.Encode(text, nil, nil))
}
