package gemini

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/antanst/go_errors"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

var (
	ErrInputTooLarge  = errors.New("input too large")
	ErrUTF8Conversion = errors.New("UTF-8 conversion error")
)

func BytesToValidUTF8(input []byte) (string, error) {
	if len(input) == 0 {
		return "", nil
	}
	const maxSize = 10 * 1024 * 1024 // 10MB
	if len(input) > maxSize {
		return "", go_errors.NewError(fmt.Errorf("%w: %d bytes (max %d)", ErrInputTooLarge, len(input), maxSize))
	}
	// remove NULL byte 0x00 (ReplaceAll accepts slices)
	inputNoNull := bytes.ReplaceAll(input, []byte{byte(0)}, []byte{})
	if utf8.Valid(inputNoNull) {
		return string(inputNoNull), nil
	}
	encodings := []transform.Transformer{
		charmap.ISO8859_1.NewDecoder(),
		charmap.ISO8859_7.NewDecoder(),
		charmap.Windows1250.NewDecoder(), // Central European
		charmap.Windows1251.NewDecoder(), // Cyrillic
		charmap.Windows1252.NewDecoder(),
		charmap.Windows1256.NewDecoder(), // Arabic
		japanese.EUCJP.NewDecoder(),      // Japanese
		korean.EUCKR.NewDecoder(),        // Korean
	}
	// First successful conversion wins.
	var lastErr error
	for _, encoding := range encodings {
		reader := transform.NewReader(bytes.NewReader(inputNoNull), encoding)
		result, err := io.ReadAll(reader)
		if err != nil {
			lastErr = err
			continue
		}
		if utf8.Valid(result) {
			return string(result), nil
		}
	}

	return "", go_errors.NewError(fmt.Errorf("%w (tried %d encodings): %w", ErrUTF8Conversion, len(encodings), lastErr))
}
