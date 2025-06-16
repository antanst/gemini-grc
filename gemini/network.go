package gemini

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	stdurl "net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"gemini-grc/common/contextlog"
	"gemini-grc/common/snapshot"
	_url "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/contextutil"
	"git.antanst.com/antanst/logging"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
)

// Visit visits a given URL using the Gemini protocol,
// and returns a populated snapshot. Any relevant errors
// when visiting the URL are stored in the snapshot;
// an error is returned only when construction of a
// snapshot was not possible (context cancellation errors,
// not a valid URL etc.)
func Visit(ctx context.Context, url string) (s *snapshot.Snapshot, err error) {
	geminiCtx := contextutil.ContextWithComponent(ctx, "network")

	s, err = snapshot.SnapshotFromURL(url, true)
	if err != nil {
		return nil, err
	}

	// Check if the context has been canceled
	if err := ctx.Err(); err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	data, err := ConnectAndGetData(geminiCtx, s.URL.String())
	if err != nil {
		s.Error = null.StringFrom(err.Error())
		return s, nil
	}

	// Check if the context has been canceled
	if err := ctx.Err(); err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	s = UpdateSnapshotWithData(*s, data)

	if !s.Error.Valid &&
		s.MimeType.Valid &&
		s.MimeType.String == "text/gemini" &&
		len(s.GemText.ValueOrZero()) > 0 {
		links := GetPageLinks(s.URL, s.GemText.String)
		if len(links) > 0 {
			s.Links = null.ValueFrom(links)
		}
	}

	return s, nil
}

// ConnectAndGetData is a context-aware version of ConnectAndGetData
// that returns the data from a GET request to a Gemini URL. It uses the context
// for cancellation, timeout, and logging.
func ConnectAndGetData(ctx context.Context, url string) ([]byte, error) {
	parsedURL, err := stdurl.Parse(url)
	if err != nil {
		return nil, xerrors.NewSimpleError(fmt.Errorf("error parsing URL: %w", err))
	}

	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "1965"
	}
	host := fmt.Sprintf("%s:%s", hostname, port)

	// Check if the context has been canceled before proceeding
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	timeoutDuration := time.Duration(config.CONFIG.ResponseTimeout) * time.Second

	// Establish the underlying TCP connection with context-based cancellation
	dialer := &net.Dialer{
		Timeout: timeoutDuration,
	}

	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Failed to establish TCP connection: %v", err)
		return nil, xerrors.NewSimpleError(err)
	}

	// Make sure we always close the connection
	defer func() {
		_ = conn.Close()
	}()

	err = conn.SetReadDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}
	err = conn.SetWriteDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	// Check if the context has been canceled before proceeding with TLS handshake
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Perform the TLS handshake
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,                 //nolint:gosec    // Accept all TLS certs, even if insecure.
		ServerName:         parsedURL.Hostname(), // SNI says we should not include port in hostname
	}

	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.SetReadDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}
	err = tlsConn.SetWriteDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	// Check if the context is done before attempting handshake
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Perform TLS handshake with regular method
	// (HandshakeContext is only available in Go 1.17+)
	err = tlsConn.Handshake()
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	// Check again if the context is done after handshake
	if err := ctx.Err(); err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	// We read `buf`-sized chunks and add data to `data`
	buf := make([]byte, 4096)
	var data []byte

	// Check if the context has been canceled before sending request
	if err := ctx.Err(); err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	// Send Gemini request to trigger server response
	// Fix for stupid server bug:
	// Some servers return 'Header: 53 No proxying to other hosts or ports!'
	// when the port is 1965 and is still specified explicitly in the URL.
	url2, _ := _url.ParseURL(url, "", true)
	_, err = tlsConn.Write([]byte(fmt.Sprintf("%s\r\n", url2.StringNoDefaultPort())))
	if err != nil {
		return nil, xerrors.NewSimpleError(err)
	}

	// Read response bytes in len(buf) byte chunks
	for {
		// Check if the context has been canceled before each read
		if err := ctx.Err(); err != nil {
			return nil, xerrors.NewSimpleError(err)
		}

		n, err := tlsConn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Response too large (max: %d bytes)", config.CONFIG.MaxResponseSize)
			return nil, xerrors.NewSimpleError(fmt.Errorf("response too large"))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Error reading data: %v", err)
			return nil, xerrors.NewSimpleError(err)
		}
	}

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Received %d bytes of data", len(data))
	return data, nil
}

// UpdateSnapshotWithData processes the raw data from a Gemini response and populates the Snapshot.
// This function is exported for use by the robotsMatch package.
func UpdateSnapshotWithData(s snapshot.Snapshot, data []byte) *snapshot.Snapshot {
	header, body, err := getHeadersAndData(data)
	if err != nil {
		s.Error = null.StringFrom(err.Error())
		return &s
	}
	code, mimeType, lang := getMimeTypeAndLang(header)

	if code != 0 {
		s.ResponseCode = null.IntFrom(int64(code))
	}
	if header != "" {
		s.Header = null.StringFrom(header)
	}
	if mimeType != "" {
		s.MimeType = null.StringFrom(mimeType)
	}
	if lang != "" {
		s.Lang = null.StringFrom(lang)
	}

	// If we've got a Gemini document, populate
	// `GemText` field, otherwise raw data goes to `Data`.
	if mimeType == "text/gemini" {
		validBody, err := BytesToValidUTF8(body)
		if err != nil {
			s.Error = null.StringFrom(err.Error())
			return &s
		}
		s.GemText = null.StringFrom(validBody)
	} else {
		s.Data = null.ValueFrom(body)
	}
	return &s
}

// Checks for a Gemini header, which is
// basically the first line of the response
// and should contain the response code,
// mimeType and language.
func getHeadersAndData(data []byte) (string, []byte, error) {
	firstLineEnds := slices.Index(data, '\n')
	if firstLineEnds == -1 {
		return "", nil, xerrors.NewSimpleError(fmt.Errorf("error parsing header"))
	}
	firstLine := string(data[:firstLineEnds])
	rest := data[firstLineEnds+1:]
	return strings.TrimSpace(firstLine), rest, nil
}

// getMimeTypeAndLang Parses code, mime type and language
// given a Gemini header.
func getMimeTypeAndLang(headers string) (int, string, string) {
	// First try to match the full format: "<code> <mimetype> [charset=<value>] [lang=<value>]"
	// The regex looks for:
	// - A number (\d+)
	// - Followed by whitespace and a mimetype ([a-zA-Z0-9/\-+]+)
	// - Optionally followed by charset and/or lang parameters in any order
	// - Only capturing the lang value, ignoring charset
	re := regexp.MustCompile(`^(\d+)\s+([a-zA-Z0-9/\-+]+)(?:(?:[\s;]+(?:charset=[^;\s]+|lang=([a-zA-Z0-9-]+)))*)\s*$`)
	matches := re.FindStringSubmatch(headers)
	if len(matches) <= 1 {
		// If full format doesn't match, try to match redirect format: "<code> <URL>"
		// This handles cases like "31 gemini://example.com"
		re := regexp.MustCompile(`^(\d+)\s+(.+)$`)
		matches := re.FindStringSubmatch(headers)
		if len(matches) <= 1 {
			// If redirect format doesn't match, try to match just a status code
			// This handles cases like "99"
			re := regexp.MustCompile(`^(\d+)\s*$`)
			matches := re.FindStringSubmatch(headers)
			if len(matches) <= 1 {
				return 0, "", ""
			}
			code, err := strconv.Atoi(matches[1])
			if err != nil {
				return 0, "", ""
			}
			return code, "", ""
		}
		code, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, "", ""
		}
		return code, "", ""
	}
	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", ""
	}
	mimeType := matches[2]
	lang := matches[3] // Will be empty string if no lang parameter was found
	return code, mimeType, lang
}
