package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"time"
)

func Visit(url string) (result *Snapshot) {
	result = &Snapshot{Timestamp: time.Now(), UID: UID()}

	// Wrap error with additional information
	defer func() {
		if result.Error != nil {
			result.Error = fmt.Errorf("[%s] Error: %w", result.URL, result.Error)
		}
	}()

	geminiUrl, err := ParseUrl(url, "")
	if err != nil {
		result.Error = err
		return result
	}
	result.URL = *geminiUrl

	LogDebug("[%s] Connecting", geminiUrl)

	// Establish a TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", geminiUrl.Hostname, geminiUrl.Port), tlsConfig)
	if err != nil {
		result.Error = err
		return result
	}
	// Defer properly: Also handle possible
	// error of conn.Close()
	defer func() {
		err := conn.Close()
		if err != nil {
			result.Error = fmt.Errorf("[%s] Closing connection error, ignoring: %w", result.URL.String(), err)
		}
	}()

	// Read data from the connection
	// TODO make timeout configurable
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024)
	var data []byte
	// Write Gemini request to get response.
	conn.Write([]byte(fmt.Sprintf("%s\r\n", geminiUrl.String())))
	// Read response bytes in len(buf) byte chunks
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if len(data) > CONFIG.maxResponseSize {
			result.Error = fmt.Errorf("Response size exceeded maximum of %d bytes", CONFIG.maxResponseSize)
			return result
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				result.Error = err
				return result
			}
		}
	}
	LogDebug("[%s] Received %d bytes", geminiUrl.String(), len(data))
	// time.Sleep(time.Duration(time.Second * 2))
	// LogDebug("[%s] Visitor finished", geminiUrl.String())
	result.Data = string(data)
	return result
}
