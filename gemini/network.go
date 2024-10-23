package gemini

import (
	"crypto/tls"
	"fmt"
	"gemini-grc/config"
	"io"
	"net"
	go_url "net/url"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/guregu/null/v5"
)

type GeminiPageData struct {
	ResponseCode int
	MimeType     string
	Lang         string
	GemText      string
	Data         []byte
}

// Resolve the URL hostname and
// check if we already have an open
// connection to this host.
// If we can connect, return a list
// of the resolved IPs.
func getHostIPAddresses(hostname string) ([]string, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return nil, err
	}
	IpPool.Lock.RLock()
	defer func() {
		IpPool.Lock.RUnlock()
	}()
	return addrs, nil
}

func ConnectAndGetData(url string) ([]byte, error) {
	parsedUrl, err := go_url.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("Could not parse URL, error %w", err)
	}
	host := parsedUrl.Host
	port := parsedUrl.Port()
	if port == "" {
		port = "1965"
		host = fmt.Sprintf("%s:%s", host, port)
	}
	// Establish the underlying TCP connection.
	dialer := &net.Dialer{
		Timeout:   time.Duration(config.CONFIG.ResponseTimeout) * time.Second,
		KeepAlive: 10 * time.Second,
	}
	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("TCP connection failed: %w", err)
	}
	// Make sure we always close the connection.
	defer func() {
		err := conn.Close()
		if err != nil {
			// Do nothing! Connection will timeout eventually if still open somehow.
		}
	}()

	// Set read and write timeouts on the TCP connection.
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(config.CONFIG.ResponseTimeout) * time.Second))
	if err != nil {
		return nil, fmt.Errorf("Error setting connection deadline: %w", err)
	}
	err = conn.SetWriteDeadline(time.Now().Add(time.Duration(config.CONFIG.ResponseTimeout) * time.Second))
	if err != nil {
		return nil, fmt.Errorf("Error setting connection deadline: %w", err)
	}

	// Perform the TLS handshake
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,           // Accept all TLS certs, even if insecure.
		ServerName:         parsedUrl.Host, // SNI
		// MinVersion:         tls.VersionTLS12, // Use a minimum TLS version. Warning breaks a lot of sites.
	}
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, fmt.Errorf("TLS handshake error: %w", err)
	}

	// We read `buf`-sized chunks and add data to `data`.
	buf := make([]byte, 4096)
	var data []byte

	// Send Gemini request to trigger server response.
	_, err = tlsConn.Write([]byte(fmt.Sprintf("%s\r\n", url)))
	if err != nil {
		return nil, fmt.Errorf("Error sending network request: %w", err)
	}
	// Read response bytes in len(buf) byte chunks
	for {
		n, err := tlsConn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			data = []byte{}
			return nil, fmt.Errorf("Response size exceeded maximum of %d bytes", config.CONFIG.MaxResponseSize)
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, fmt.Errorf("Network error: %s", err)
			}
		}
	}
	return data, nil
}

// Connect to given URL, using the Gemini protocol.
// Mutate given Snapshot with the data or the error.
func Visit(s *Snapshot) {
	data, err := ConnectAndGetData(s.URL.String())
	if err != nil {
		s.Error = null.StringFrom(err.Error())
		return
	}
	pageData, err := processData(data)
	if err != nil {
		s.Error = null.StringFrom(err.Error())
		return
	}
	s.ResponseCode = null.IntFrom(int64(pageData.ResponseCode))
	s.MimeType = null.StringFrom(pageData.MimeType)
	s.Lang = null.StringFrom(pageData.Lang)
	if pageData.GemText != "" {
		s.GemText = null.StringFrom(string(pageData.GemText))
	}
	if pageData.Data != nil {
		s.Data = null.ValueFrom(pageData.Data)
	}
	return
}

// Update given snapshot with the
// Gemini header data: response code,
// mime type and lang (optional)
func processData(data []byte) (*GeminiPageData, error) {
	headers, body, err := getHeadersAndData(data)
	if err != nil {
		return nil, err
	}
	code, mimeType, lang := getMimeTypeAndLang(headers)
	geminiError := checkGeminiStatusCode(code)
	if geminiError != nil {
		return nil, geminiError
	}
	pageData := GeminiPageData{
		ResponseCode: code,
		MimeType:     mimeType,
		Lang:         lang,
	}
	// If we've got a Gemini document, populate
	// `GemText` field, otherwise raw data goes to `Data`.
	if mimeType == "text/gemini" {
		validBody, err := EnsureValidUTF8(body)
		if err != nil {
			return nil, fmt.Errorf("UTF-8 error: %w", err)
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
func getHeadersAndData(data []byte) (firstLine string, rest []byte, err error) {
	firstLineEnds := slices.Index(data, '\n')
	if firstLineEnds == -1 {
		return "", nil, fmt.Errorf("Could not parse response header")
	}
	firstLine = string(data[:firstLineEnds])
	rest = data[firstLineEnds+1:]
	return string(firstLine), rest, nil
}

// Parses code, mime type and language
// from a Gemini header.
// Examples:
// `20 text/gemini lang=en` (code, mimetype, lang)
// `20 text/gemini` (code, mimetype)
// `31 gemini://redirected.to/other/site` (code)
func getMimeTypeAndLang(headers string) (code int, mimeType string, lang string) {
	// Regex that parses code, mimetype & lang
	re := regexp.MustCompile(`^(\d+)\s+([a-zA-Z0-9/\-+]+)(?:[;\s]+(lang=([a-zA-Z0-9-]+)))?\s*$`)
	matches := re.FindStringSubmatch(headers)
	if matches == nil || len(matches) <= 1 {
		// Try to get code at least.
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
	mimeType = matches[2]
	lang = matches[4]
	return code, mimeType, lang
}
