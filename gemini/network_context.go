package gemini

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	stdurl "net/url"
	"time"

	"gemini-grc/common/contextlog"
	commonErrors "gemini-grc/common/errors"
	"gemini-grc/common/snapshot"
	_url "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/contextutil"
	"gemini-grc/logging"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
)

// Visit visits a given URL using the Gemini protocol.
func Visit(ctx context.Context, url string) (s *snapshot.Snapshot, err error) {
	geminiCtx := contextutil.ContextWithComponent(ctx, "gemini")

	contextlog.LogDebugWithContext(geminiCtx, logging.GetSlogger(), "Visiting Gemini URL: %s", url)

	s, err = snapshot.SnapshotFromURL(url, true)
	if err != nil {
		contextlog.LogErrorWithContext(geminiCtx, logging.GetSlogger(), "Failed to create snapshot from URL: %v", err)
		return nil, err
	}

	defer func() {
		if err == nil {
			return
		}
		// GeminiError and HostError should
		// be stored in the snapshot.
		if commonErrors.IsHostError(err) {
			contextlog.LogInfoWithContext(geminiCtx, logging.GetSlogger(), "Host error: %v", err)
			s.Error = null.StringFrom(err.Error())
			err = nil
			return
		} else if IsGeminiError(err) {
			contextlog.LogInfoWithContext(geminiCtx, logging.GetSlogger(), "Gemini error: %v", err)
			s.Error = null.StringFrom(err.Error())
			s.Header = null.StringFrom(errors.Unwrap(err).(*GeminiError).Header)
			s.ResponseCode = null.IntFrom(int64(errors.Unwrap(err).(*GeminiError).Code))
			err = nil
			return
		}
	}()

	// Check if the context has been canceled
	if err := ctx.Err(); err != nil {
		return s, err
	}

	data, err := ConnectAndGetDataWithContext(geminiCtx, s.URL.String())
	if err != nil {
		return s, err
	}

	// Check if the context has been canceled
	if err := ctx.Err(); err != nil {
		return s, err
	}

	s, err = ProcessData(*s, data)
	if err != nil {
		return s, err
	}

	if isGeminiCapsule(s) {
		links := GetPageLinks(s.URL, s.GemText.String)
		if len(links) > 0 {
			s.Links = null.ValueFrom(links)
		}
	}

	contextlog.LogDebugWithContext(geminiCtx, logging.GetSlogger(), "Successfully visited URL: %s (Code: %d)", url, s.ResponseCode.ValueOrZero())
	return s, nil
}

// ConnectAndGetDataWithContext is a context-aware version of ConnectAndGetData
// that returns the data from a GET request to a Gemini URL. It uses the context
// for cancellation, timeout, and logging.
func ConnectAndGetDataWithContext(ctx context.Context, url string) ([]byte, error) {
	// Parse the URL
	parsedURL, err := stdurl.Parse(url)
	if err != nil {
		return nil, xerrors.NewError(fmt.Errorf("error parsing URL: %w", err), 0, "", false)
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

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Connecting to %s", host)

	timeoutDuration := time.Duration(config.CONFIG.ResponseTimeout) * time.Second

	// Establish the underlying TCP connection with context-based cancellation
	dialer := &net.Dialer{
		Timeout: timeoutDuration,
	}

	// Use DialContext to allow cancellation via context
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Failed to establish TCP connection: %v", err)
		return nil, commonErrors.NewHostError(err)
	}

	// Make sure we always close the connection
	defer func() {
		_ = conn.Close()
	}()

	// Set read and write timeouts on the TCP connection
	err = conn.SetReadDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, commonErrors.NewHostError(err)
	}
	err = conn.SetWriteDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, commonErrors.NewHostError(err)
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
		return nil, commonErrors.NewHostError(err)
	}
	err = tlsConn.SetWriteDeadline(time.Now().Add(timeoutDuration))
	if err != nil {
		return nil, commonErrors.NewHostError(err)
	}

	// Check if the context is done before attempting handshake
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Perform TLS handshake with regular method
	// (HandshakeContext is only available in Go 1.17+)
	err = tlsConn.Handshake()
	if err != nil {
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "TLS handshake failed: %v", err)
		return nil, commonErrors.NewHostError(err)
	}

	// Check again if the context is done after handshake
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// We read `buf`-sized chunks and add data to `data`
	buf := make([]byte, 4096)
	var data []byte

	// Check if the context has been canceled before sending request
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Send Gemini request to trigger server response
	// Fix for stupid server bug:
	// Some servers return 'Header: 53 No proxying to other hosts or ports!'
	// when the port is 1965 and is still specified explicitly in the URL.
	url2, _ := _url.ParseURL(url, "", true)
	_, err = tlsConn.Write([]byte(fmt.Sprintf("%s\r\n", url2.StringNoDefaultPort())))
	if err != nil {
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to send request: %v", err)
		return nil, commonErrors.NewHostError(err)
	}

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Request sent, reading response")

	// Read response bytes in len(buf) byte chunks
	for {
		// Check if the context has been canceled before each read
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		n, err := tlsConn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Response too large (max: %d bytes)", config.CONFIG.MaxResponseSize)
			return nil, commonErrors.NewHostError(fmt.Errorf("response too large"))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Error reading data: %v", err)
			return nil, commonErrors.NewHostError(err)
		}
	}

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Received %d bytes of data", len(data))
	return data, nil
}

// ProcessDataWithContext is a context-aware version of ProcessData that processes
// the raw data from a Gemini response and populates the Snapshot.
func ProcessDataWithContext(ctx context.Context, s snapshot.Snapshot, data []byte) (*snapshot.Snapshot, error) {
	// Create a processing-specific context with the "process" component
	processCtx := contextutil.ContextWithComponent(ctx, "process")

	contextlog.LogDebugWithContext(processCtx, logging.GetSlogger(), "Processing Gemini response data (%d bytes)", len(data))

	header, body, err := getHeadersAndData(data)
	if err != nil {
		contextlog.LogErrorWithContext(processCtx, logging.GetSlogger(), "Failed to extract headers: %v", err)
		return &s, err
	}

	code, mimeType, lang := getMimeTypeAndLang(header)
	contextlog.LogDebugWithContext(processCtx, logging.GetSlogger(), "Response code: %d, MimeType: %s, Lang: %s", code, mimeType, lang)

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
			contextlog.LogErrorWithContext(processCtx, logging.GetSlogger(), "UTF-8 validation failed: %v", err)
			return nil, err
		}
		s.GemText = null.StringFrom(validBody)
		contextlog.LogDebugWithContext(processCtx, logging.GetSlogger(), "Processed gemtext content (%d characters)", len(validBody))
	} else {
		s.Data = null.ValueFrom(body)
		contextlog.LogDebugWithContext(processCtx, logging.GetSlogger(), "Stored binary data (%d bytes)", len(body))
	}

	return &s, nil
}
