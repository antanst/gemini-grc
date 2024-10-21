package gemini

import (
	"crypto/tls"
	"fmt"
	"gemini-grc/config"
	"io"
	"net"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/guregu/null/v5"
)

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

// Connect to given URL, using the Gemini protocol.
// Return a Snapshot with the data or the error.
// Any errors are stored within the snapshot.
func Visit(s *Snapshot) {
	// Establish the underlying TCP connection.
	host := fmt.Sprintf("%s:%d", s.Host, s.URL.Port)
	dialer := &net.Dialer{
		Timeout:   time.Duration(config.CONFIG.ResponseTimeout) * time.Second, // Set the overall connection timeout
		KeepAlive: 30 * time.Second,
	}
	conn, err := dialer.Dial("tcp", host)
	if err != nil {
		s.Error = null.StringFrom(fmt.Sprintf("TCP connection failed: %v", err))
		return
	}
	// Make sure we always close the connection.
	defer func() {
		err := conn.Close()
		if err != nil {
			s.Error = null.StringFrom(fmt.Sprintf("Error closing connection: %s", err))
		}
	}()

	// Set read and write timeouts on the TCP connection.
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(config.CONFIG.ResponseTimeout) * time.Second))
	if err != nil {
		s.Error = null.StringFrom(fmt.Sprintf("Error setting connection deadline: %s", err))
		return
	}
	err = conn.SetWriteDeadline(time.Now().Add(time.Duration(config.CONFIG.ResponseTimeout) * time.Second))
	if err != nil {
		s.Error = null.StringFrom(fmt.Sprintf("Error setting connection deadline: %s", err))
		return
	}

	// Perform the TLS handshake
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,           // Accept all TLS certs, even if insecure.
		ServerName:         s.URL.Hostname, // SNI
		// MinVersion:         tls.VersionTLS12, // Use a minimum TLS version. Warning breaks a lot of sites.
	}
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		s.Error = null.StringFrom(fmt.Sprintf("TLS handshake error: %v", err))
		return
	}

	// We read `buf`-sized chunks and add data to `data`.
	buf := make([]byte, 4096)
	var data []byte

	// Send Gemini request to trigger server response.
	_, err = tlsConn.Write([]byte(fmt.Sprintf("%s\r\n", s.URL.String())))
	if err != nil {
		s.Error = null.StringFrom(fmt.Sprintf("Error sending network request: %s", err))
		return
	}
	// Read response bytes in len(buf) byte chunks
	for {
		n, err := tlsConn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			data = []byte{}
			s.Error = null.StringFrom(fmt.Sprintf("Response size exceeded maximum of %d bytes", config.CONFIG.MaxResponseSize))
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				s.Error = null.StringFrom(fmt.Sprintf("Network error: %s", err))
				return
			}
		}
	}
	// Great, response data received.
	err = processResponse(s, data)
	if err != nil {
		s.Error = null.StringFrom(err.Error())
	}
	return
}

// Update given snapshot with the
// Gemini header data: response code,
// mime type and lang (optional)
func processResponse(snapshot *Snapshot, data []byte) error {
	headers, body, err := getHeadersAndData(data)
	if err != nil {
		return err
	}
	code, mimeType, lang := getMimeTypeAndLang(headers)
	geminiError := checkGeminiStatusCode(code)
	if geminiError != nil {
		return geminiError
	}
	snapshot.ResponseCode = null.IntFrom(int64(code))
	snapshot.MimeType = null.StringFrom(mimeType)
	snapshot.Lang = null.StringFrom(lang)
	// If we've got a Gemini document, populate
	// `GemText` field, otherwise raw data goes to `Data`.
	if mimeType == "text/gemini" {
		validBody, err := EnsureValidUTF8(body)
		if err != nil {
			return fmt.Errorf("UTF-8 error: %w", err)
		}
		snapshot.GemText = null.StringFrom(string(validBody))
	} else {
		snapshot.Data = null.ValueFrom(body)
	}
	return nil
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
