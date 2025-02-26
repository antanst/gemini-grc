package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

type fatal interface {
	Fatal() bool
}

func IsFatal(err error) bool {
	var e fatal
	ok := As(err, &e)
	return ok && e.Fatal()
}

func As(err error, target any) bool {
	return errors.As(err, target)
}

func Is(err, target error) bool {
	return errors.Is(err, target)
}

func Unwrap(err error) error {
	return errors.Unwrap(err)
}

type Error struct {
	Err   error
	Stack string
	fatal bool
}

func (e *Error) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%v\n", e.Err))
	sb.WriteString(fmt.Sprintf("Stack Trace:\n%s", e.Stack))
	return sb.String()
}

func (e *Error) Fatal() bool {
	return e.fatal
}

func (e *Error) Unwrap() error {
	return e.Err
}

func NewError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already of our own
	// Error type, so we don't add stack twice.
	var asError *Error
	if errors.As(err, &asError) {
		return err
	}

	// has the stack trace
	var stack strings.Builder
	buf := make([]uintptr, 50)
	n := runtime.Callers(2, buf)
	frames := runtime.CallersFrames(buf[:n])

	// Format the stack trace
	for {
		frame, more := frames.Next()
		// Skip runtime and standard library frames
		if !strings.Contains(frame.File, "runtime/") {
			stack.WriteString(fmt.Sprintf("\t%s:%d - %s\n", frame.File, frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}

	return &Error{
		Err:   err,
		Stack: stack.String(),
	}
}

func NewFatalError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already of our own
	// Error type.
	var asError *Error
	if errors.As(err, &asError) {
		asError.fatal = true // Set fatal even for existing Error types
		return err
	}
	err2 := NewError(err)
	err2.(*Error).fatal = true
	return err2
}
