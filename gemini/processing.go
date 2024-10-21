package gemini

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func EnsureValidUTF8(input []byte) (string, error) {
	// Remove NULL byte 0x00
	inputNoNull := bytes.ReplaceAll(input, []byte{0}, nil)
	isValidUTF8 := utf8.Valid(inputNoNull)
	if !isValidUTF8 {
		encodings := []transform.Transformer{
			charmap.ISO8859_1.NewDecoder(),   // First try ISO8859-1
			charmap.Windows1252.NewDecoder(), // Then try Windows-1252, etc
			// TODO: Try more encodings?
		}
		for _, encoding := range encodings {
			reader := transform.NewReader(bytes.NewReader(inputNoNull), encoding)
			result, err := io.ReadAll(reader)
			if err != nil {
				return "", fmt.Errorf("UTF-8 error: %w", err)
			}
			return string(result), nil
		}
	}
	return string(inputNoNull), nil
}
