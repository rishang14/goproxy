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
