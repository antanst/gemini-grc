package gemini

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func BytesToValidUTF8(input []byte) (string, error) {
	// Remove NULL byte 0x00 (ReplaceAll accepts slices)
	inputNoNull := bytes.ReplaceAll(input, []byte{byte(0)}, []byte{})
	isValidUTF8 := utf8.Valid(inputNoNull)
	if isValidUTF8 {
		return string(inputNoNull), nil
	}
	encodings := []transform.Transformer{
		charmap.ISO8859_1.NewDecoder(),   // First try ISO8859-1
		charmap.Windows1252.NewDecoder(), // Then try Windows-1252, etc
		// TODO: Try more encodings?
	}
	// First successful conversion wins.
	for _, encoding := range encodings {
		reader := transform.NewReader(bytes.NewReader(inputNoNull), encoding)
		result, err := io.ReadAll(reader)
		if err == nil {
			return string(result), nil
		}
	}
	return "", fmt.Errorf("UTF-8 error: %w", err)
}
