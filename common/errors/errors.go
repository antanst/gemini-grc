package errors

import (
	"fmt"

	"gemini-grc/errors"
)

// HostError is an error encountered while
// visiting a host, and should be recorded
// to the snapshot.
type HostError struct {
	Err error
}

func (e *HostError) Error() string {
	return e.Err.Error()
}

func (e *HostError) Unwrap() error {
	return e.Err
}

func NewHostError(err error) error {
	return &HostError{Err: err}
}

func IsHostError(err error) bool {
	if err == nil {
		return false
	}
	var asError *HostError
	return errors.As(err, &asError)
}

// Sentinel errors used for their string message primarily.
// Do not use them by themselves, to be embedded to HostError.
var (
	ErrBlacklistMatch = fmt.Errorf("black list match")
	ErrRobotsMatch    = fmt.Errorf("robots match")
)
