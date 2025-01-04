package gemini

import (
	"crypto/tls"
	"errors"
	"fmt"
	"gemini-grc/common"
	"io"
	"net"
	gourl "net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"gemini-grc/config"
	"gemini-grc/logging"
	"github.com/guregu/null/v5"
)

type PageData struct {
	ResponseCode   int
	ResponseHeader string
	MimeType       string
	Lang           string
	GemText        string
	Data           []byte
}

// Resolve the URL hostname and
// check if we already have an open
// connection to this host.
// If we can connect, return a list
// of the resolved IPs.
func getHostIPAddresses(hostname string) ([]string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil, fmt.Errorf("%w:%w", common.ErrNetworkDNS, err)
	}
	IPPool.Lock.RLock()
	defer func() {
		IPPool.Lock.RUnlock()
	}()
	return addrs, nil
}

func ConnectAndGetData(url string) ([]byte, error) {
	parsedURL, err := gourl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrURLParse, err)
	}
	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "1965"
	}
	host := fmt.Sprintf("%s:%s", hostname, port)
	// Establish the underlying TCP connection.
	dialer := &net.Dialer{
		Timeout: time.Duration(config.CONFIG.ResponseTimeout) * time.Second,
	}
	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrNetwork, err)
	}
	// Make sure we always close the connection.
	defer func() {
		// No need to handle error:
		// Connection will time out eventually if still open somehow.
		_ = conn.Close()
	}()

	// Set read and write timeouts on the TCP connection.
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(config.CONFIG.ResponseTimeout) * time.Second))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrNetworkSetConnectionDeadline, err)
	}
	err = conn.SetWriteDeadline(time.Now().Add(time.Duration(config.CONFIG.ResponseTimeout) * time.Second))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrNetworkSetConnectionDeadline, err)
	}

	// Perform the TLS handshake
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,                 //nolint:gosec    // Accept all TLS certs, even if insecure.
		ServerName:         parsedURL.Hostname(), // SNI says we should not include port in hostname
		// MinVersion:         tls.VersionTLS12, // Use a minimum TLS version. Warning breaks a lot of sites.
	}
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrNetworkTLS, err)
	}

	// We read `buf`-sized chunks and add data to `data`.
	buf := make([]byte, 4096)
	var data []byte

	// Send Gemini request to trigger server response.
	// Fix for stupid server bug:
	// Some servers return 'Header: 53 No proxying to other hosts or ports!'
	// when the port is 1965 and is still specified explicitly in the URL.
	_url, _ := common.ParseURL(url, "")
	_, err = tlsConn.Write([]byte(fmt.Sprintf("%s\r\n", _url.StringNoDefaultPort())))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", common.ErrNetworkCannotWrite, err)
	}
	// Read response bytes in len(buf) byte chunks
	for {
		n, err := tlsConn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			return nil, fmt.Errorf("%w: %v", common.ErrNetworkResponseSizeExceededMax, config.CONFIG.MaxResponseSize)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("%w: %w", common.ErrNetwork, err)
		}
	}
	return data, nil
}

// Visit given URL, using the Gemini protocol.
// Mutates given Snapshot with the data.
// In case of error, we store the error string
// inside snapshot and return the error.
func Visit(s *common.Snapshot) (err error) {
	// Don't forget to also store error
	// response code (if we have one)
	// and header
	defer func() {
		if err != nil {
			s.Error = null.StringFrom(err.Error())
			if errors.As(err, new(*common.GeminiError)) {
				s.Header = null.StringFrom(err.(*common.GeminiError).Header)
				s.ResponseCode = null.IntFrom(int64(err.(*common.GeminiError).Code))
			}
		}
	}()
	s.Timestamp = null.TimeFrom(time.Now())
	data, err := ConnectAndGetData(s.URL.String())
	if err != nil {
		return err
	}
	pageData, err := processData(data)
	if err != nil {
		return err
	}
	s.Header = null.StringFrom(pageData.ResponseHeader)
	s.ResponseCode = null.IntFrom(int64(pageData.ResponseCode))
	s.MimeType = null.StringFrom(pageData.MimeType)
	s.Lang = null.StringFrom(pageData.Lang)
	if pageData.GemText != "" {
		s.GemText = null.StringFrom(pageData.GemText)
	}
	if pageData.Data != nil {
		s.Data = null.ValueFrom(pageData.Data)
	}
	return nil
}

// processData returne results from
// parsing Gemini header data:
// Code, mime type and lang (optional)
// Returns error if header was invalid
func processData(data []byte) (*PageData, error) {
	header, body, err := getHeadersAndData(data)
	if err != nil {
		return nil, err
	}
	code, mimeType, lang := getMimeTypeAndLang(header)
	logging.LogDebug("Header: %s", strings.TrimSpace(header))
	if code != 20 {
		return nil, common.NewErrGeminiStatusCode(code, header)
	}

	pageData := PageData{
		ResponseCode:   code,
		ResponseHeader: header,
		MimeType:       mimeType,
		Lang:           lang,
	}
	// If we've got a Gemini document, populate
	// `GemText` field, otherwise raw data goes to `Data`.
	if mimeType == "text/gemini" {
		validBody, err := BytesToValidUTF8(body)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", common.ErrUTF8Parse, err)
		}
		pageData.GemText = validBody
	} else {
		pageData.Data = body
	}
	return &pageData, nil
}

// Checks for a Gemini header, which is
// basically the first line of the response
// and should contain the response code,
// mimeType and language.
func getHeadersAndData(data []byte) (string, []byte, error) {
	firstLineEnds := slices.Index(data, '\n')
	if firstLineEnds == -1 {
		return "", nil, common.ErrGeminiResponseHeader
	}
	firstLine := string(data[:firstLineEnds])
	rest := data[firstLineEnds+1:]
	return firstLine, rest, nil
}

// Parses code, mime type and language
// from a Gemini header.
// Examples:
// `20 text/gemini lang=en` (code, mimetype, lang)
// `20 text/gemini` (code, mimetype)
// `31 gemini://redirected.to/other/site` (code)
func getMimeTypeAndLang(headers string) (int, string, string) {
	// Regex that parses code, mimetype & optional charset/lang parameters
	re := regexp.MustCompile(`^(\d+)\s+([a-zA-Z0-9/\-+]+)(?:[;\s]+(?:(?:charset|lang)=([a-zA-Z0-9-]+)))?\s*$`)
	matches := re.FindStringSubmatch(headers)
	if matches == nil || len(matches) <= 1 {
		// Try to get code at least
		re := regexp.MustCompile(`^(\d+)\s+`)
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
	mimeType := matches[2]
	param := matches[3] // This will capture either charset or lang value
	return code, mimeType, param
}
