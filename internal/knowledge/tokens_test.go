package knowledge

import (
	"testing"
)

func TestNewTokenCounter_Success(t *testing.T) {
	counter, err := NewTokenCounter()

	if err != nil {
		t.Fatalf("NewTokenCounter() failed: %v", err)
	}

	if counter == nil {
		t.Fatal("NewTokenCounter() returned nil counter")
	}
}

func TestTokenCounter_CountTokens_SimpleText(t *testing.T) {
	counter, err := NewTokenCounter()
	if err != nil {
		t.Fatalf("NewTokenCounter() failed: %v", err)
	}

	text := "Hello world"
	count := counter.CountTokens(text)

	// "Hello world" should be ~2-3 tokens
	if count < 1 || count > 5 {
		t.Errorf("CountTokens(%q) = %d, expected 2-3 tokens", text, count)
	}
}

func TestTokenCounter_CountTokens_EmptyString(t *testing.T) {
	counter, err := NewTokenCounter()
	if err != nil {
		t.Fatalf("NewTokenCounter() failed: %v", err)
	}

	count := counter.CountTokens("")

	if count != 0 {
		t.Errorf("CountTokens(\"\") = %d, expected 0", count)
	}
}

func TestTokenCounter_CountTokens_LongText(t *testing.T) {
	counter, err := NewTokenCounter()
	if err != nil {
		t.Fatalf("NewTokenCounter() failed: %v", err)
	}

	// Approximate 100 tokens worth of text
	text := "This is a test sentence that contains multiple words and should result in a reasonable token count. " +
		"We want to verify that the token counter can handle longer pieces of text accurately. " +
		"The cl100k_base encoding should give us consistent results."

	count := counter.CountTokens(text)

	// Should be roughly 50-70 tokens
	if count < 40 || count > 80 {
		t.Errorf("CountTokens(long text) = %d, expected ~50-70 tokens", count)
	}
}

func TestTokenCounter_Fallback(t *testing.T) {
	// Test fallback when encoder is nil
	counter := &TokenCounter{encoder: nil}

	text := "Hello world with some text"
	count := counter.CountTokens(text)

	// Fallback uses len(text)/4 approximation
	expected := len(text) / 4
	if count != expected {
		t.Errorf("Fallback CountTokens() = %d, expected %d", count, expected)
	}
}
