package errors_test

import (
	"errors"
	"fmt"
	"testing"

	"gemini-grc/gemini"
)

func TestErrGemini(t *testing.T) {
	t.Parallel()
	err := gemini.NewGeminiError(50, "50 server error")
	if !errors.As(err, new(*gemini.GeminiError)) {
		t.Errorf("TestErrGemini fail")
	}
}

func TestErrGeminiWrapped(t *testing.T) {
	t.Parallel()
	err := gemini.NewGeminiError(50, "50 server error")
	errWrapped := fmt.Errorf("%w wrapped", err)
	if !errors.As(errWrapped, new(*gemini.GeminiError)) {
		t.Errorf("TestErrGeminiWrapped fail")
	}
}

func TestIsGeminiError(t *testing.T) {
	t.Parallel()
	err1 := gemini.NewGeminiError(50, "50 server error")
	if !gemini.IsGeminiError(err1) {
		t.Errorf("TestGeminiError fail #1")
	}
	wrappedErr1 := fmt.Errorf("wrapped %w", err1)
	if !gemini.IsGeminiError(wrappedErr1) {
		t.Errorf("TestGeminiError fail #2")
	}
}
