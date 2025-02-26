package gemini

import (
	"fmt"

	"gemini-grc/errors"
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

func NewGeminiError(code int, header string) error {
	var msg string
	switch {
	case code >= 10 && code < 20:
		msg = "needs input"
	case code >= 30 && code < 40:
		msg = "redirect"
	case code >= 40 && code < 50:
		msg = "bad request"
	case code >= 50 && code < 60:
		msg = "server error"
	case code >= 60 && code < 70:
		msg = "TLS error"
	default:
		msg = "unexpected Status code"
	}
	return &GeminiError{
		Msg:    msg,
		Code:   code,
		Header: header,
	}
}

func IsGeminiError(err error) bool {
	if err == nil {
		return false
	}
	var asError *GeminiError
	return errors.As(err, &asError)
}
