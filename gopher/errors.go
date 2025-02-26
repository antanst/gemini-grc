package gopher

import (
	"gemini-grc/errors"
)

// GopherError is an error encountered while
// visiting a Gopher host, and is only for
// Gopher errors (item type indicator 3).
type GopherError struct {
	Err error
}

func (e *GopherError) Error() string {
	return e.Err.Error()
}

func (e *GopherError) Unwrap() error {
	return e.Err
}

func NewGopherError(err error) error {
	return &GopherError{Err: err}
}

func IsGopherError(err error) bool {
	if err == nil {
		return false
	}
	var asError *GopherError
	return errors.As(err, &asError)
}
