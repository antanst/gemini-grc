package gemini

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrGemini(t *testing.T) {
	t.Parallel()
	err := NewGeminiError(50, "50 server error")
	if !errors.As(err, new(*GeminiError)) {
		t.Errorf("TestErrGemini fail")
	}
}

func TestErrGeminiWrapped(t *testing.T) {
	t.Parallel()
	err := NewGeminiError(50, "50 server error")
	errWrapped := fmt.Errorf("%w wrapped", err)
	if !errors.As(errWrapped, new(*GeminiError)) {
		t.Errorf("TestErrGeminiWrapped fail")
	}
}

func TestIsGeminiError(t *testing.T) {
	t.Parallel()
	err1 := NewGeminiError(50, "50 server error")
	if !IsGeminiError(err1) {
		t.Errorf("TestGeminiError fail #1")
	}
	wrappedErr1 := fmt.Errorf("wrapped %w", err1)
	if !IsGeminiError(wrappedErr1) {
		t.Errorf("TestGeminiError fail #2")
	}
}
