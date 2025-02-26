package errors

import (
	"errors"
	"fmt"
	"testing"
)

type CustomError struct {
	Err error
}

func (e *CustomError) Error() string { return e.Err.Error() }

func IsCustomError(err error) bool {
	var asError *CustomError
	return errors.As(err, &asError)
}

func TestWrapping(t *testing.T) {
	t.Parallel()
	originalErr := errors.New("original error")
	err1 := NewError(originalErr)
	if !errors.Is(err1, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	if !Is(err1, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	unwrappedErr := errors.Unwrap(err1)
	if !errors.Is(unwrappedErr, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	if !Is(unwrappedErr, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	unwrappedErr = Unwrap(err1)
	if !errors.Is(unwrappedErr, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	if !Is(unwrappedErr, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	wrappedErr := fmt.Errorf("wrapped: %w", originalErr)
	if !errors.Is(wrappedErr, originalErr) {
		t.Errorf("original error is not wrapped")
	}
	if !Is(wrappedErr, originalErr) {
		t.Errorf("original error is not wrapped")
	}
}

func TestNewError(t *testing.T) {
	t.Parallel()
	originalErr := &CustomError{errors.New("err1")}
	if !IsCustomError(originalErr) {
		t.Errorf("TestNewError fail #1")
	}
	err1 := NewError(originalErr)
	if !IsCustomError(err1) {
		t.Errorf("TestNewError fail #2")
	}
	wrappedErr1 := fmt.Errorf("wrapped %w", err1)
	if !IsCustomError(wrappedErr1) {
		t.Errorf("TestNewError fail #3")
	}
	unwrappedErr1 := Unwrap(wrappedErr1)
	if !IsCustomError(unwrappedErr1) {
		t.Errorf("TestNewError fail #4")
	}
}

func TestIsFatal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "simple non-fatal error",
			err:  fmt.Errorf("regular error"),
			want: false,
		},
		{
			name: "direct fatal error",
			err:  NewFatalError(fmt.Errorf("fatal error")),
			want: true,
		},
		{
			name: "non-fatal Error type",
			err:  NewError(fmt.Errorf("non-fatal error")),
			want: false,
		},
		{
			name: "wrapped fatal error - one level",
			err:  fmt.Errorf("outer: %w", NewFatalError(fmt.Errorf("inner fatal"))),
			want: true,
		},
		{
			name: "wrapped fatal error - two levels",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("middle: %w",
					NewFatalError(fmt.Errorf("inner fatal")))),
			want: true,
		},
		{
			name: "wrapped fatal error - three levels",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("middle1: %w",
					fmt.Errorf("middle2: %w",
						NewFatalError(fmt.Errorf("inner fatal"))))),
			want: true,
		},
		{
			name: "multiple wrapped errors - non-fatal",
			err: fmt.Errorf("outer: %w",
				fmt.Errorf("middle: %w",
					fmt.Errorf("inner: %w",
						NewError(fmt.Errorf("base error"))))),
			want: false,
		},
		{
			name: "wrapped non-fatal Error type",
			err:  fmt.Errorf("outer: %w", NewError(fmt.Errorf("inner"))),
			want: false,
		},
		{
			name: "wrapped basic error",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("inner")),
			want: false,
		},
		{
			name: "fatal error wrapping fatal error",
			err:  NewFatalError(NewFatalError(fmt.Errorf("double fatal"))),
			want: true,
		},
		{
			name: "fatal error wrapping non-fatal Error",
			err:  NewFatalError(NewError(fmt.Errorf("mixed"))),
			want: true,
		},
		{
			name: "non-fatal Error wrapping fatal error",
			err:  NewError(NewFatalError(fmt.Errorf("mixed"))),
			want: true,
		},
		{
			name: "Error wrapping Error",
			err:  NewError(NewError(fmt.Errorf("double wrapped"))),
			want: false,
		},
		{
			name: "wrapped nil error",
			err:  fmt.Errorf("outer: %w", nil),
			want: false,
		},
		{
			name: "fatal wrapping nil",
			err:  NewFatalError(nil),
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsFatal(tt.err)
			if got != tt.want {
				t.Errorf("IsFatal() = %v, want %v", got, tt.want)
				if tt.err != nil {
					t.Errorf("Error was: %v", tt.err)
				}
			}
		})
	}
}
