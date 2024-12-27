package gemini

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrGemini(t *testing.T) {
	t.Parallel()
	err := NewErrGeminiStatusCode(50, "50 server error")
	if !errors.As(err, new(*GeminiError)) {
		t.Errorf("TestErrGemini fail")
	}
}

func TestErrGeminiWrapped(t *testing.T) {
	t.Parallel()
	err := NewErrGeminiStatusCode(50, "50 server error")
	errWrapped := fmt.Errorf("%w wrapped", err)
	if !errors.As(errWrapped, new(*GeminiError)) {
		t.Errorf("TestErrGeminiWrapped fail")
	}
}
