package gemini

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	stdurl "net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	errors2 "gemini-grc/common/errors"
	"gemini-grc/common/snapshot"
	_url "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/logging"

	"github.com/antanst/go_errors"
	"github.com/guregu/null/v5"
)

// Visit given URL, using the Gemini protocol.
// Mutates given Snapshot with the data.
// In case of error, we store the error string
// inside snapshot and return the error.
func Visit(url string) (s *snapshot.Snapshot, err error) {
	s, err = snapshot.SnapshotFromURL(url, true)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			// GeminiError and HostError should
			// be stored in the snapshot. Other
			// errors are returned.
			if errors2.IsHostError(err) {
				s.Error = null.StringFrom(err.Error())
				err = nil
			} else if IsGeminiError(err) {
				s.Error = null.StringFrom(err.Error())
				s.Header = null.StringFrom(go_errors.Unwrap(err).(*GeminiError).Header)
				s.ResponseCode = null.IntFrom(int64(go_errors.Unwrap(err).(*GeminiError).Code))
				err = nil
			} else {
				s = nil
			}
		}
	}()

	data, err := ConnectAndGetData(s.URL.String())
	if err != nil {
		return s, err
	}

	s, err = processData(*s, data)
	if err != nil {
		return s, err
	}

	if isGeminiCapsule(s) {
		links := GetPageLinks(s.URL, s.GemText.String)
		if len(links) > 0 {
			logging.LogDebug("Found %d links", len(links))
			s.Links = null.ValueFrom(links)
		}
	}
	return s, nil
}

func ConnectAndGetData(url string) ([]byte, error) {
	parsedURL, err := stdurl.Parse(url)
	if err != nil {
		return nil, go_errors.NewError(err)
	}
	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "1965"
	}
	host := fmt.Sprintf("%s:%s", hostname, port)
	timeoutDuration := time.Duration(config.CONFIG.ResponseTimeout) * time.Second
	// Establish the underlying TCP connection.
	dialer := &net.Dialer{
		Timeout: timeoutDuration,
	}
	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		return nil, errors2.NewHostError(err)
	}
	// Make sure we always close the connection.
	defer func() {
		_ = conn.Close()
	}()

	// Set read and write timeouts on the TCP connection.
	err = conn.SetReadDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, errors2.NewHostError(err)
	}
	err = conn.SetWriteDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, errors2.NewHostError(err)
	}

	// Perform the TLS handshake
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,                 //nolint:gosec    // Accept all TLS certs, even if insecure.
		ServerName:         parsedURL.Hostname(), // SNI says we should not include port in hostname
		// MinVersion:         tls.VersionTLS12, // Use a minimum TLS version. Warning breaks a lot of sites.
	}
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.SetReadDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, errors2.NewHostError(err)
	}
	err = tlsConn.SetWriteDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, errors2.NewHostError(err)
	}
	err = tlsConn.Handshake()
	if err != nil {
		return nil, errors2.NewHostError(err)
	}

	// We read `buf`-sized chunks and add data to `data`.
	buf := make([]byte, 4096)
	var data []byte

	// Send Gemini request to trigger server response.
	// Fix for stupid server bug:
	// Some servers return 'Header: 53 No proxying to other hosts or ports!'
	// when the port is 1965 and is still specified explicitly in the URL.
	url2, _ := _url.ParseURL(url, "", true)
	_, err = tlsConn.Write([]byte(fmt.Sprintf("%s\r\n", url2.StringNoDefaultPort())))
	if err != nil {
		return nil, errors2.NewHostError(err)
	}
	// Read response bytes in len(buf) byte chunks
	for {
		n, err := tlsConn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			return nil, errors2.NewHostError(err)
		}
		if err != nil {
			if go_errors.Is(err, io.EOF) {
				break
			}
			return nil, errors2.NewHostError(err)
		}
	}
	return data, nil
}

func processData(s snapshot.Snapshot, data []byte) (*snapshot.Snapshot, error) {
	header, body, err := getHeadersAndData(data)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		s.GemText = null.StringFrom(validBody)
	} else {
		s.Data = null.ValueFrom(body)
	}
	return &s, nil
}

// Checks for a Gemini header, which is
// basically the first line of the response
// and should contain the response code,
// mimeType and language.
func getHeadersAndData(data []byte) (string, []byte, error) {
	firstLineEnds := slices.Index(data, '\n')
	if firstLineEnds == -1 {
		return "", nil, errors2.NewHostError(fmt.Errorf("error parsing header"))
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
	if matches == nil || len(matches) <= 1 {
		// If full format doesn't match, try to match redirect format: "<code> <URL>"
		// This handles cases like "31 gemini://example.com"
		re := regexp.MustCompile(`^(\d+)\s+(.+)$`)
		matches := re.FindStringSubmatch(headers)
		if matches == nil || len(matches) <= 1 {
			// If redirect format doesn't match, try to match just a status code
			// This handles cases like "99"
			re := regexp.MustCompile(`^(\d+)\s*$`)
			matches := re.FindStringSubmatch(headers)
			if matches == nil || len(matches) <= 1 {
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

func isGeminiCapsule(s *snapshot.Snapshot) bool {
	return !s.Error.Valid && s.MimeType.Valid && s.MimeType.String == "text/gemini"
}
