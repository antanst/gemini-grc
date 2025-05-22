package gemini

import (
	"errors"
	"fmt"
)

// GeminiError is used to represent
// Gemini network protocol errors only.
// Should be recorded to the snapshot.
// See https://geminiprotocol.net/docs/protocol-specification.gmi
type GeminiError struct {
	Msg    string
	Code   int
	Header string
}

func (e *GeminiError) Error() string {
	return fmt.Sprintf("gemini error: code %d %s", e.Code, e.Msg)
}

func (e *GeminiError) String() string {
	return e.Error()
}

// NewGeminiError creates a new GeminiError based on the status code and header.
// Status codes are based on the Gemini protocol specification:
// - 1x: Input required
// - 2x: Success (not handled as errors)
// - 3x: Redirect
// - 4x: Temporary failure
// - 5x: Permanent failure
// - 6x: Client certificate required/rejected
func NewGeminiError(code int, header string) error {
	var msg string
	switch {
	case code >= 10 && code < 20:
		msg = fmt.Sprintf("input required: %s", header)
	case code >= 30 && code < 40:
		msg = fmt.Sprintf("redirect: %s", header)
	case code >= 40 && code < 50:
		msg = fmt.Sprintf("request failed: %s", header)
	case code >= 50 && code < 60:
		msg = fmt.Sprintf("server error: %s", header)
	case code >= 60 && code < 70:
		msg = fmt.Sprintf("TLS error: %s", header)
	default:
		msg = fmt.Sprintf("unexpected status code %d: %s", code, header)
	}
	return &GeminiError{
		Msg:    msg,
		Code:   code,
		Header: header,
	}
}

// IsGeminiError checks if the given error is a GeminiError.
func IsGeminiError(err error) bool {
	if err == nil {
		return false
	}
	var asError *GeminiError
	return errors.As(err, &asError)
}
