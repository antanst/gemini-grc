package text

import "strings"

// RemoveNullChars removes all null characters from the input string.
func RemoveNullChars(input string) string {
	// Replace all null characters with an empty string
	return strings.ReplaceAll(input, "\u0000", "")
}
