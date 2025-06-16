package gopher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	stdurl "net/url"
	"time"
	"unicode/utf8"

	"gemini-grc/common/contextlog"
	commonErrors "gemini-grc/common/errors"
	"gemini-grc/common/linkList"
	"gemini-grc/common/snapshot"
	"gemini-grc/common/text"
	_url "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/contextutil"
	"git.antanst.com/antanst/logging"
	"git.antanst.com/antanst/xerrors"
	"github.com/guregu/null/v5"
)

// VisitWithContext is a context-aware version of Visit that visits
// a given URL using the Gopher protocol. It uses the context for
// cancellation, timeout, and logging.
func VisitWithContext(ctx context.Context, url string) (*snapshot.Snapshot, error) {
	// Create a gopher-specific context with the "gopher" component
	gopherCtx := contextutil.ContextWithComponent(ctx, "gopher")

	if !config.CONFIG.GopherEnable {
		contextlog.LogDebugWithContext(gopherCtx, logging.GetSlogger(), "Gopher protocol is disabled")
		return nil, nil
	}

	s, err := snapshot.SnapshotFromURL(url, true)
	if err != nil {
		contextlog.LogErrorWithContext(gopherCtx, logging.GetSlogger(), "Failed to create snapshot from URL: %v", err)
		return nil, err
	}

	// Check if the context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := connectAndGetDataWithContext(gopherCtx, url)
	if err != nil {
		contextlog.LogDebugWithContext(gopherCtx, logging.GetSlogger(), "Error: %s", err.Error())
		if IsGopherError(err) || commonErrors.IsHostError(err) {
			s.Error = null.StringFrom(err.Error())
			return s, nil
		}
		return nil, err
	}

	// Check if the context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	isValidUTF8 := utf8.ValidString(string(data))
	if isValidUTF8 {
		s.GemText = null.StringFrom(text.RemoveNullChars(string(data)))
		contextlog.LogDebugWithContext(gopherCtx, logging.GetSlogger(), "Response is valid UTF-8 text (%d bytes)", len(data))
	} else {
		s.Data = null.ValueFrom(data)
		contextlog.LogDebugWithContext(gopherCtx, logging.GetSlogger(), "Response is binary data (%d bytes)", len(data))
	}

	if !isValidUTF8 {
		return s, nil
	}

	responseError := checkForError(string(data))
	if responseError != nil {
		contextlog.LogErrorWithContext(gopherCtx, logging.GetSlogger(), "Gopher server returned error: %v", responseError)
		s.Error = null.StringFrom(responseError.Error())
		return s, nil
	}

	// Extract links from the response
	links := getGopherPageLinks(string(data))
	linkURLs := linkList.LinkList(make([]_url.URL, len(links)))

	for i, link := range links {
		linkURL, err := _url.ParseURL(link, "", true)
		if err == nil {
			linkURLs[i] = *linkURL
		}
	}

	if len(links) != 0 {
		s.Links = null.ValueFrom(linkURLs)
		contextlog.LogDebugWithContext(gopherCtx, logging.GetSlogger(), "Found %d links in gopher page", len(links))
	}

	contextlog.LogDebugWithContext(gopherCtx, logging.GetSlogger(), "Successfully visited Gopher URL: %s", url)
	return s, nil
}

// connectAndGetDataWithContext is a context-aware version of connectAndGetData
func connectAndGetDataWithContext(ctx context.Context, url string) ([]byte, error) {
	parsedURL, err := stdurl.Parse(url)
	if err != nil {
		return nil, xerrors.NewError(fmt.Errorf("error parsing URL: %w", err), 0, "", false)
	}

	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "70"
	}
	host := fmt.Sprintf("%s:%s", hostname, port)

	// Use the context's deadline if it has one, otherwise use the config timeout
	var timeoutDuration time.Duration
	deadline, ok := ctx.Deadline()
	if ok {
		timeoutDuration = time.Until(deadline)
	} else {
		timeoutDuration = time.Duration(config.CONFIG.ResponseTimeout) * time.Second
	}

	// Check if the context is canceled
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Dialing %s", host)

	// Establish the underlying TCP connection with context-based cancellation
	dialer := &net.Dialer{
		Timeout: timeoutDuration,
	}

	// Use DialContext to allow cancellation via context
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to connect: %v", err)
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

	// We read `buf`-sized chunks and add data to `data`
	buf := make([]byte, 4096)
	var data []byte

	// Check if the context is canceled before sending request
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Send Gopher request to trigger server response
	payload := constructPayloadFromPath(parsedURL.Path)
	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Sending request with payload: %s", payload)
	_, err = conn.Write([]byte(fmt.Sprintf("%s\r\n", payload)))
	if err != nil {
		contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Failed to send request: %v", err)
		return nil, commonErrors.NewHostError(err)
	}

	// Read response bytes in len(buf) byte chunks
	for {
		// Check if the context is canceled before each read
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Error reading data: %v", err)
			return nil, commonErrors.NewHostError(err)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			contextlog.LogErrorWithContext(ctx, logging.GetSlogger(), "Response too large (max: %d bytes)", config.CONFIG.MaxResponseSize)
			return nil, commonErrors.NewHostError(fmt.Errorf("response exceeded max"))
		}
	}

	contextlog.LogDebugWithContext(ctx, logging.GetSlogger(), "Received %d bytes", len(data))
	return data, nil
}
