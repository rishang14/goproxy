package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/rishang14/goproxy/internal/httpx"
)

type helloHandler struct{}

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("Error listening", err)
	}
	defer listener.Close()
	router := httpx.NewRouter()
	router.Handle("GET", "/hello", helloHandler{})

	fmt.Println("TCP Server started at port:8080")
	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Println("Error while connecting", err)
			continue
		}

		go handleConnection(conn, router)

	}
}

// it will accept the incoming connection and send everything to readrequest
func handleConnection(conn net.Conn, router *httpx.Router) {

	defer conn.Close()

	defer func() {

		if r := recover(); r != nil {
			log.Printf("panic serving %v: %v\n%s", conn.RemoteAddr(), r, debug.Stack())
			httpx.WriteErrorFor(conn, httpx.ErrMalformedRequest)

		}
	}()

	reader := bufio.NewReader(conn)
	for {
		err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.Printf("Timeout Error : %s", err)
			return
		}

		req, err := httpx.ReadRequest(reader)
		if err != nil {
			// goodbye: client closed, or idle past the deadline
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrDeadlineExceeded) {
				return
			}
			// a real bad request: tell the client why
			httpx.WriteErrorFor(conn, err)
			return
		}
		keepAlive := shouldKeepAlive(req.Version, req.Headers)
		httpx.Serve(conn, router, req, keepAlive)
		if !keepAlive {
			return
		}
	}

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

func (h helloHandler) ServeHTTP(w httpx.ResponseWriter, r *httpx.Request) {
	w.WriteBody([]byte("hello from " + r.Path))
}
