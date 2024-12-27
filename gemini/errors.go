package gemini

import (
	"errors"
	"fmt"
)

type GeminiError struct {
	Msg    string
	Code   int
	Header string
}

func (e *GeminiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Msg, e.Header)
}

func NewErrGeminiStatusCode(code int, header string) error {
	var msg string
	switch {
	case code >= 10 && code < 20:
		msg = "needs input"
	case code >= 30 && code < 40:
		msg = "redirect"
	case code >= 40 && code < 50:
		msg = "bad request"
	case code >= 50 && code < 60:
		msg = "server error"
	case code >= 60 && code < 70:
		msg = "TLS error"
	default:
		msg = "unexpected status code"
	}
	return &GeminiError{
		Msg:    msg,
		Code:   code,
		Header: header,
	}
}

var (
	ErrGeminiRobotsParse      = errors.New("gemini robots.txt parse error")
	ErrGeminiRobotsDisallowed = errors.New("gemini robots.txt disallowed")
	ErrGeminiResponseHeader   = errors.New("gemini response header error")
	ErrGeminiRedirect         = errors.New("gemini redirection error")
	ErrGeminiLinkLineParse    = errors.New("gemini link line parse error")

	ErrURLParse     = errors.New("URL parse error")
	ErrURLNotGemini = errors.New("not a Gemini URL")
	ErrURLDecode    = errors.New("URL decode error")
	ErrUTF8Parse    = errors.New("UTF-8 parse error")
	ErrTextParse    = errors.New("text parse error")

	ErrNetwork                        = errors.New("network error")
	ErrNetworkDNS                     = errors.New("network DNS error")
	ErrNetworkTLS                     = errors.New("network TLS error")
	ErrNetworkSetConnectionDeadline   = errors.New("network error - cannot set connection deadline")
	ErrNetworkCannotWrite             = errors.New("network error - cannot write")
	ErrNetworkResponseSizeExceededMax = errors.New("network error - response size exceeded maximum size")

	ErrDatabase = errors.New("database error")
)

// We could have used a map for speed, but
// we would lose ability to check wrapped
// errors via errors.Is().

var errGemini *GeminiError

var knownErrors = []error{ //nolint:gochecknoglobals
	errGemini,
	ErrGeminiLinkLineParse,
	ErrGeminiRobotsParse,
	ErrGeminiRobotsDisallowed,
	ErrGeminiResponseHeader,
	ErrGeminiRedirect,

	ErrURLParse,
	ErrURLDecode,
	ErrUTF8Parse,
	ErrTextParse,

	ErrNetwork,
	ErrNetworkDNS,
	ErrNetworkTLS,
	ErrNetworkSetConnectionDeadline,
	ErrNetworkCannotWrite,
	ErrNetworkResponseSizeExceededMax,

	ErrDatabase,
}

func IsKnownError(err error) bool {
	for _, known := range knownErrors {
		if errors.Is(err, known) {
			return true
		}
	}
	return errors.As(err, new(*GeminiError))
}
