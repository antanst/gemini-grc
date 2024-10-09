package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"time"
)

func Visit(url string) (snapshot *Snapshot, err error) {
	snapshot = &Snapshot{Timestamp: time.Now(), UID: UID()}

	geminiUrl, err := ParseUrl(url, "")
	if err != nil {
		snapshot.Error = fmt.Errorf("[%s] %w", url, err)
		return snapshot, nil
	}
	snapshot.URL = *geminiUrl

	LogDebug("[%s] Connecting", geminiUrl)

	// Establish a TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", geminiUrl.Hostname, geminiUrl.Port), tlsConfig)
	if err != nil {
		snapshot.Error = err
		return snapshot, nil
	}
	// Defer properly: Also handle possible
	// error of conn.Close()
	defer func() {
		err := conn.Close()
		if err != nil {
			snapshot.Error = fmt.Errorf("[%s] Closing connection error, ignoring: %w", snapshot.URL.String(), err)
		}
	}()

	// Read data from the connection
	conn.SetReadDeadline(time.Now().Add(time.Duration(CONFIG.responseTimeout) * time.Second))
	buf := make([]byte, 4096)
	var data []byte

	// Write Gemini request to get response.
	//	paths := []string{"/", ".", ""}
	//	if slices.Contains(paths, geminiUrl.Path) || strings.HasSuffix(geminiUrl.Path, "gmi") {
	conn.Write([]byte(fmt.Sprintf("%s\r\n", geminiUrl.String())))
	//	}

	// Read response bytes in len(buf) byte chunks
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > CONFIG.maxResponseSize {
			snapshot.Error = fmt.Errorf("[%s] Response size exceeded maximum of %d bytes", url, CONFIG.maxResponseSize)
			return snapshot, nil
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				snapshot.Error = fmt.Errorf("[%s] %w", url, err)
				return snapshot, nil
			}
		}
	}
	LogDebug("[%s] Received %d bytes", geminiUrl.String(), len(data))
	err = processResponse(snapshot, data)
	if err != nil {
		snapshot.Error = fmt.Errorf("%w", err)
	}
	return snapshot, nil
}

func processResponse(snapshot *Snapshot, data []byte) error {
	headers, body, err := getHeadersAndData(data)
	if err != nil {
		return err
	}
	code, mimeType, lang := getMimeTypeAndLang(headers)
	snapshot.ResponseCode, snapshot.MimeType, snapshot.Lang, snapshot.Data = code, mimeType, lang, body
	if mimeType == "text/gemini" {
		snapshot.GemText = string(body)
	}
	return nil
}

func getHeadersAndData(data []byte) (string, []byte, error) {
	firstLineEnds := slices.Index(data, '\n')
	if firstLineEnds == -1 {
		return "", nil, fmt.Errorf("Could not parse response header")
	}
	firstLine := data[:firstLineEnds]
	rest := data[firstLineEnds+1:]
	return string(firstLine), rest, nil
}

func getMimeTypeAndLang(headers string) (int, string, string) {
	re := regexp.MustCompile(`^(\d+)\s+([a-zA-Z0-9/\-+]+)[;\s]+(lang=([a-zA-Z0-9-]+))?`)
	matches := re.FindStringSubmatch(headers)
	if matches == nil || len(matches) <= 1 {
		return 0, "", ""
	}
	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", ""
	}
	mimeType := matches[2]
	lang := matches[4]
	return code, mimeType, lang
}
