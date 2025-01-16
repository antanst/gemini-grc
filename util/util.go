package util

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"runtime/debug"
)

func PrintStackAndPanic(err error) {
	fmt.Printf("PANIC Error %s Stack trace:\n%s", err, debug.Stack())
	panic("PANIC")
}

// SecureRandomInt returns a cryptographically secure random integer in the range [0,max).
// Panics if max <= 0 or if there's an error reading from the system's secure
// random number generator.
func SecureRandomInt(max int) int {
	// Convert max to *big.Int for crypto/rand operations
	maxBig := big.NewInt(int64(max))

	// Generate random number
	n, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		PrintStackAndPanic(fmt.Errorf("could not generate a random integer between 0 and %d", max))
	}

	// Convert back to int
	return int(n.Int64())
}

func PrettyJson(data string) string {
	marshalled, _ := json.MarshalIndent(data, "", "  ")
	return fmt.Sprintf("%s\n", marshalled)
}

// GetLinesMatchingRegex returns all lines that match given regex
func GetLinesMatchingRegex(input string, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(input, -1)
	return matches
}
