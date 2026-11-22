package httpx

import (
	"bufio"
	"io"
	"log"
	"net/textproto"
	"strconv"
	"strings"
)

const (
	maxBodyBytes   = 1 << 20 // 1 MiB — matches nginx's default
	maxHeaderBytes = 1 << 16 // 64 KiB total across all header lines
	maxHeaderCount = 100     // number of header lines
)

type Request struct {
	Method  string
	Target  string
	Version string
	Headers map[string][]string
	Body    []byte
}

// Receive the  incoming request parse it  and set the header
// and it will also read the content body length etc a
func ReadRequest(r *bufio.Reader) (*Request, error) {
	headerCount := 0
	headerBytes := 0
	headers := make(map[string][]string)
	requestLine, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	method, target, version, err := ParseRequest(requestLine)
	if err != nil {

		return nil, err
	}
	for {
		header, err := r.ReadString('\n')
		headerCount++
		headerBytes += len(header)

		if headerCount > maxHeaderCount || headerBytes > maxHeaderBytes {
			return nil, ErrHeaderLimitExceeded
		}
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(header) == "" {
			break
		}

		name, value, found := strings.Cut(header, ":")

		if !found {
			log.Printf("malformed header line: %q", header)
			return nil, ErrMalformedRequest
		}

		key := canonicalKey(strings.TrimSpace(name))
		headers[key] = append(headers[key], strings.TrimSpace(value))

	}

	var body []byte

	if ck, ok := headers["Content-Length"]; ok {
		if len(ck) > 1 {

			return nil, ErrMalformedRequest
		}
		n, err := strconv.Atoi(ck[0])
		if err != nil {
			return nil, ErrMalformedRequest
		}
		if n < 0 {

			return nil, ErrMalformedRequest
		}
		if n > maxBodyBytes {
			return nil, ErrContentTooLarge
		}
		body = make([]byte, n)
		got, err := io.ReadFull(r, body)
		if err != nil {
			log.Printf("body need %d got %d bytes err %v", len(body), got, err)
			return nil, ErrMalformedRequest
		}
	}
	return &Request{Method: method, Target: target, Version: version, Headers: headers, Body: body}, nil
}

func canonicalKey(key string) string {
	value := textproto.CanonicalMIMEHeaderKey(key)
	return value
}
