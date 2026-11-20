package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/rishang14/goproxy/internal/httpx"
)

const (
	maxBodyBytes   = 1 << 20 // 1 MiB — matches nginx's default
	maxHeaderBytes = 1 << 16 // 64 KiB total across all header lines
	maxHeaderCount = 100     // number of header lines
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Error listening", err)
	}
	defer listener.Close()

	fmt.Println("TCP Server started at port:8080")

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println("Error while connecting", err)
			continue
		}

		go handleConnection(conn)

	}
}

// it will accept the input from the incoming request parse it  and set the header and all and  parse it
// and it will also read the content body and everythig
func handleConnection(conn net.Conn) {

	defer conn.Close()

	defer func() {

		if r := recover(); r != nil {
			log.Printf("panic serving %v: %v\n%s", conn.RemoteAddr(), r, debug.Stack())
			httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)

		}
	}()

	reader := bufio.NewReader(conn)
	for {
		headerCount := 0
		headerBytes := 0
		headers := make(map[string][]string)
		err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.Printf("Timeout Error : %s", err)
			return
		}
		requestLine, err := reader.ReadString('\n')

		if err != nil {
			log.Printf("Reading error %v", err)
			return
		}

		method, target, version, err := httpx.ParseRequest(requestLine)
		if err != nil {
			httpx.WriteErrorFor(conn, err)
			return
		}
		fmt.Printf("Method is %s  target is %s  version is %s \n", method, target, version)
		for {
			header, err := reader.ReadString('\n')
			headerCount++
			headerBytes += len(header)

			if headerCount > maxHeaderCount || headerBytes > maxHeaderBytes {
				httpx.WriteErrorFor(conn, httpx.ErrHeaderLimitExceeded)
				return
			}
			if err != nil {
				log.Printf("Reading error %v", err)
				return
			}
			if strings.TrimSpace(header) == "" {
				break
			}

			name, value, found := strings.Cut(header, ":")

			if !found {
				log.Printf("malformed header line: %q", header)
				httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)
				return
			}

			key := canonicalKey(strings.TrimSpace(name))
			headers[key] = append(headers[key], strings.TrimSpace(value))
			fmt.Printf("header key %s  and value is %s  \n", key, headers[key])

		}

		var body []byte

		if ck, ok := headers["Content-Length"]; ok {
			if len(ck) > 1 {
				httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)
				return
			}
			n, err := strconv.Atoi(ck[0])
			if err != nil {
				httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)
				return
			}
			if n < 0 {
				httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)
				return
			}
			if n > maxBodyBytes {
				httpx.WriteErrorFor(conn, httpx.ErrContentTooLarge)
				return
			}
			body = make([]byte, n)
			got, err := io.ReadFull(reader, body)
			if err != nil {
				log.Printf("body need %d got %d bytes err %v", len(body), got, err)
				httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)
				return
			}
		}
		keepAlive := shouldKeepAlive(version, headers)
		httpx.WriteResponse(conn, 200, "OK", body, keepAlive)
		if !keepAlive {
			return
		}
	}

}

func canonicalKey(key string) string {
	value := textproto.CanonicalMIMEHeaderKey(key)
	return value
}

// checks request HTTP/1.1 defaults to keeping it open; HTTP/1.0 defaults
func shouldKeepAlive(version string, headers map[string][]string) bool {

	val := headers["Connection"]

	if len(val) == 0 {
		return version == "HTTP/1.1"
	}

	if version == "HTTP/1.0" {
		return strings.EqualFold(val[0], "keep-alive")
	}

	return !strings.EqualFold(val[0], "close")
}
