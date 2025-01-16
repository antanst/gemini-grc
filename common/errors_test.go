package common_test

import (
	"errors"
	"fmt"
	"gemini-grc/common"
	"testing"
)

func TestErrGemini(t *testing.T) {
	t.Parallel()
	err := common.NewErrGeminiStatusCode(50, "50 server error")
	if !errors.As(err, new(*common.GeminiError)) {
		t.Errorf("TestErrGemini fail")
	}
}

func TestErrGeminiWrapped(t *testing.T) {
	t.Parallel()
	err := common.NewErrGeminiStatusCode(50, "50 server error")
	errWrapped := fmt.Errorf("%w wrapped", err)
	if !errors.As(errWrapped, new(*common.GeminiError)) {
		t.Errorf("TestErrGeminiWrapped fail")
	}
}
