package gemini

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"gemini-grc/config"
	"git.antanst.com/antanst/xerrors"
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

	maxSize := config.CONFIG.MaxResponseSize
	if maxSize == 0 {
		maxSize = 1024 * 1024 // Default 1MB for tests
	}
	if len(input) > maxSize {
		return "", xerrors.NewError(fmt.Errorf("BytesToValidUTF8: %w: %d bytes (max %d)", ErrInputTooLarge, len(input), maxSize), 0, "", false)
	}

	// Always remove NULL bytes first (before UTF-8 validity check)
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

	// Still invalid Unicode. Try some encodings to convert to.
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

	return "", xerrors.NewError(fmt.Errorf("BytesToValidUTF8: %w (tried %d encodings): %w", ErrUTF8Conversion, len(encodings), lastErr), 0, "", false)
}
