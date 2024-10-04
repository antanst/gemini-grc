package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"time"
)

func Visit(url string) (result *Result) {
	result = &Result{}

	// Wrap error with additional information
	defer func() {
		if result.error != nil {
			result.error = fmt.Errorf("[%s] Error: %w", result.url, result.error)
		}
	}()

	geminiUrl, err := ParseUrl(url, "")
	if err != nil {
		result.error = err
		return result
	}
	result.url = *geminiUrl

	LogInfo("[%s] Dialing", geminiUrl.String())

	// Establish a TLS connection
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", geminiUrl.hostname, geminiUrl.port), tlsConfig)
	if err != nil {
		result.error = err
		return result
	}
	defer conn.Close()

	// Read data from the connection
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
		if err != nil {
			if err == io.EOF {
				break
			} else {
				result.error = err
				return result
			}
		}
	}
	LogInfo("[%s] Received %d bytes", geminiUrl.String(), len(data))
	// time.Sleep(time.Duration(time.Second * 2))
	// LogDebug("[%s] Visitor finished", geminiUrl.String())
	result.data = string(data)
	return result
}
