package gopher

import (
	"fmt"
	"io"
	"net"
	stdurl "net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	errors2 "gemini-grc/common/errors"
	"gemini-grc/common/linkList"
	"gemini-grc/common/snapshot"
	"gemini-grc/common/text"
	_url "gemini-grc/common/url"
	"gemini-grc/config"
	"gemini-grc/logging"
	"github.com/antanst/go_errors"
	"github.com/guregu/null/v5"
)

// References:
// RFC 1436 https://www.rfc-editor.org/rfc/rfc1436.html

// The default port for Gopher is 70.
// Originally Gopher used ASCII or
// ISO-8859-1, now others use UTF-8.
// In any case, just converting to UTF-8
// will work. If not, we bail.

// Here's the complete list of Gopher item type indicators (prefixes):
//
// `0` - Plain Text File
// `1` - Directory/Menu
// `2` - CSO Phone Book Server
// `3` - Error Message
// `4` - BinHexed Macintosh File
// `5` - DOS Binary Archive
// `6` - UNIX uuencoded File
// `7` - Index/Search Server
// `8` - Telnet Session
// `9` - Binary File
// `+` - Mirror/Redundant Server
// `g` - GIF Image
// `I` - Image File (non-GIF)
// `T` - TN3270 Session
// `i` - Informational Message (menu line)
// `h` - HTML File
// `s` - Sound/Music File
// `d` - Document File
// `w` - WHOIS Service
// `;` - Document File with Alternative View
// `<` - Video File
// `M` - MIME File (mail message or similar)
// `:` - Bitmap Image
// `c` - Calendar File
// `p` - PostScript File

// The most commonly used ones are `0` (text), `1` (directory), `i` (info), and `3` (error).
// The original Gopher protocol only specified types 0-9, `+`, `g`, `I`, and `T`.
// The others were added by various implementations and extensions over time.

// Error methodology:
// HostError for DNS/network errors
// GopherError for network/gopher errors
// NewError for other errors
// NewFatalError for other fatal errors

func Visit(url string) (*snapshot.Snapshot, error) {
	s, err := snapshot.SnapshotFromURL(url, false)
	if err != nil {
		return nil, err
	}

	data, err := connectAndGetData(url)
	if err != nil {
		logging.LogDebug("Error: %s", err.Error())
		if IsGopherError(err) || errors2.IsHostError(err) {
			s.Error = null.StringFrom(err.Error())
			return s, nil
		}
		return nil, err
	}

	isValidUTF8 := utf8.ValidString(string(data))
	if isValidUTF8 {
		s.GemText = null.StringFrom(text.RemoveNullChars(string(data)))
	} else {
		s.Data = null.ValueFrom(data)
	}

	if !isValidUTF8 {
		return s, nil
	}

	responseError := checkForError(string(data))
	if responseError != nil {
		s.Error = null.StringFrom(responseError.Error())
		return s, nil
	}

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
	}

	return s, nil
}

func connectAndGetData(url string) ([]byte, error) {
	parsedURL, err := stdurl.Parse(url)
	if err != nil {
		return nil, go_errors.NewError(err)
	}

	hostname := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "70"
	}
	host := fmt.Sprintf("%s:%s", hostname, port)
	timeoutDuration := time.Duration(config.CONFIG.ResponseTimeout) * time.Second
	// Establish the underlying TCP connection.
	dialer := &net.Dialer{
		Timeout: timeoutDuration,
	}
	logging.LogDebug("Dialing %s", host)
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

	// We read `buf`-sized chunks and add data to `data`.
	buf := make([]byte, 4096)
	var data []byte

	// Send Gopher request to trigger server response.
	payload := constructPayloadFromPath(parsedURL.Path)
	_, err = conn.Write([]byte(fmt.Sprintf("%s\r\n", payload)))
	if err != nil {
		return nil, errors2.NewHostError(err)
	}
	// Read response bytes in len(buf) byte chunks
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err != nil {
			if go_errors.Is(err, io.EOF) {
				break
			}
			return nil, errors2.NewHostError(err)
		}
		if len(data) > config.CONFIG.MaxResponseSize {
			return nil, errors2.NewHostError(fmt.Errorf("response exceeded max"))
		}
	}
	logging.LogDebug("Got %d bytes", len(data))
	return data, nil
}

func constructPayloadFromPath(urlpath string) string {
	// remove Gopher item type in URL from payload, if one.
	re := regexp.MustCompile(`^/[\w]/.*`)
	payloadWithoutItemtype := urlpath
	if re.Match([]byte(urlpath)) {
		payloadWithoutItemtype = strings.Join(strings.Split(urlpath, "/")[2:], "/")
	}
	if !strings.HasPrefix(payloadWithoutItemtype, "/") {
		payloadWithoutItemtype = fmt.Sprintf("/%s", payloadWithoutItemtype)
	}
	return payloadWithoutItemtype
}

func checkForError(utfData string) error {
	lines := strings.Split(strings.TrimSpace(utfData), "\n")
	var firstLine string
	if len(lines) > 0 {
		firstLine = lines[0]
	} else {
		return nil
	}
	if strings.HasPrefix(firstLine, "3") {
		split := strings.Split(firstLine, "\t")
		return NewGopherError(fmt.Errorf("gopher error: %s", strings.TrimSpace(split[0])))
	}
	return nil
}

func getGopherPageLinks(content string) []string {
	var links []string

	lines := strings.Split(strings.TrimSpace(content), "\n")

	for _, line := range lines {
		if line == "" || line == "." {
			continue
		}

		if len(line) < 1 {
			continue
		}

		itemType := line[0]
		if itemType == 'i' {
			continue
		}

		parts := strings.SplitN(line[1:], "\t", 4)
		if len(parts) < 3 {
			continue
		}

		selector := strings.TrimSpace(parts[1])
		host := strings.TrimSpace(parts[2])

		if host == "" {
			continue
		}

		// Handle HTML links first
		if itemType == 'h' && strings.HasPrefix(selector, "URL:") {
			if url := strings.TrimSpace(selector[4:]); url != "" {
				links = append(links, url)
			}
			continue
		}

		// For gopher links, build URL carefully
		var url strings.Builder

		// Protocol and host:port
		url.WriteString("gopher://")
		url.WriteString(host)
		url.WriteString(":")
		if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
			url.WriteString(strings.TrimSpace(parts[3]))
		} else {
			url.WriteString("70")
		}

		// Path: always /type + selector
		url.WriteString("/")
		url.WriteString(string(itemType))
		if strings.HasPrefix(selector, "/") {
			url.WriteString(selector)
		} else {
			url.WriteString("/" + selector)
		}

		links = append(links, url.String())
	}

	return links
}
