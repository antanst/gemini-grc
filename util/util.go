package util

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
)

// SecureRandomInt returns a cryptographically secure random integer in the range [0,max).
// Panics if max <= 0 or if there's an error reading from the system's secure
// random number generator.
func SecureRandomInt(max int) int {
	// Convert max to *big.Int for crypto/rand operations
	maxBig := big.NewInt(int64(max))

	// Generate random number
	n, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		panic(fmt.Errorf("could not generate a random integer between 0 and %d", max))
	}

	// Convert back to int
	return int(n.Int64())
}

func PrettifyJson(data string) string {
	marshalled, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("%s\n", marshalled)
}

// GetLinesMatchingRegex returns all lines that match given regex
func GetLinesMatchingRegex(input string, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(input, -1)
	return matches
}

// Filter applies a predicate function to each element in a slice and returns a new slice
// containing only the elements for which the predicate returns true.
// Type parameter T allows this function to work with slices of any type.
func Filter[T any](slice []T, f func(T) bool) []T {
	filtered := make([]T, 0)
	for _, v := range slice {
		if f(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// Map applies a function to each element in a slice and returns a new slice
// containing the results.
// Type parameters T and R allow this function to work with different input and output types.
func Map[T any, R any](slice []T, f func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = f(v)
	}
	return result
}
