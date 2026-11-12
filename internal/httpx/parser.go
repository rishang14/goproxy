package httpx

import (
	"fmt"
	"strings"
)

type Error struct {
	Status int
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("httpx: %d %s", e.Status, e.Reason)
}

var (
	ErrUnsupportedMethod  = &Error{Status: 405, Reason: "Method Not Allowed"}
	ErrUnsupportedVersion = &Error{Status: 505, Reason: "HTTP Version Not Supported"}
	ErrMalformedRequest   = &Error{Status: 400, Reason: "Bad Request"}
)

// this function basically parse the request line and tell it is valid or not
func ParseRequest(val string) (string, string, string, error) {
	method, parts, found := strings.Cut(val, " ")

	if !found {
		return " ", " ", " ", ErrMalformedRequest
	}

	if !checkMethod(method) {
		return " ", " ", " ", ErrUnsupportedMethod
	}

	target, version, found := strings.Cut(parts, " ")

	if !found {
		return " ", " ", " ", ErrMalformedRequest
	}

	version = strings.TrimSpace(version)

	if version != "HTTP/1.1" && version != "HTTP/1.0" {
		return "", "", "", ErrUnsupportedVersion
	}

	return method, target, version, nil
}

func checkMethod(method string) bool {
	switch method {
	case "GET", "PUT", "POST", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
