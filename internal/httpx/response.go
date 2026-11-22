package httpx

import (
	"errors"
	"fmt"
	"io"
)

// WriteError sends a minimal HTTP/1.1 error response with no body.
func WriteError(w io.Writer, status int, reason string) error {
	resp := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\n"+
			"Content-Length: 0\r\n"+
			"Connection: close\r\n"+
			"\r\n",
		status, reason,
	)
	_, err := w.Write([]byte(resp))
	return err
}

// WriteErrorFor writes the response matching err. Errors from this
// package carry their own status; anything else becomes a 400.
func WriteErrorFor(w io.Writer, err error) error {
	var hErr *Error
	if errors.As(err, &hErr) {
		return WriteError(w, hErr.Status, hErr.Reason)
	}
	return WriteError(w, 400, "Bad Request")
}

// WriteResponse write to a client
func WriteResponse(w io.Writer, status int, reason string, body []byte, connection bool, version string) error {
	val := ""

	if connection {
		val = "keep-alive"
	} else {
		val = "close"
	}
	resp := fmt.Sprintf(
		"%s %d %s\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: %s\r\n"+
			"\r\n",
		version, status, reason, len(body), val,
	)
	out := append([]byte(resp), body...)
	_, err := w.Write(out)
	return err
}

func StatusText(status int) string {
	switch status {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	}
	return ""
}
