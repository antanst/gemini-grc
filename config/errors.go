package config

import "fmt"

// ValidationError represents a config validation error
type ValidationError struct {
	Param  string
	Value  string
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("invalid value '%s' for %s: %s", e.Value, e.Param, e.Reason)
}
