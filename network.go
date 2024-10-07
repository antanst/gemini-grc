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
			result.Error = fmt.Errorf("[%s] Error: %w", result.Url, result.Error)
		}
	}()

	geminiUrl, err := ParseUrl(url, "")
	if err != nil {
		result.Error = err
		return result
	}
	result.Url = *geminiUrl

	LogInfo("[%s] Dialing", geminiUrl)

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
			result.Error = fmt.Errorf("[%s] Closing connection error, ignoring: %w", result.Url.String(), err)
		}
	}()

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
				result.Error = err
				return result
			}
		}
	}
	LogInfo("[%s] Received %d bytes", geminiUrl.String(), len(data))
	// time.Sleep(time.Duration(time.Second * 2))
	// LogDebug("[%s] Visitor finished", geminiUrl.String())
	result.Data = string(data)
	return result
}
