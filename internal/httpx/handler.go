package httpx

import (
	"io"
)

type ResponseWriter interface {
	WriteResponse(status int)
	WriteBody(b []byte) (int, error)
}

type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

var _ ResponseWriter = (*response)(nil)

type response struct {
	status int
	body   []byte
}

func (r *response) WriteResponse(status int) {
	r.status = status
}

func (r *response) WriteBody(body []byte) (int, error) {
	r.body = append(r.body, body...)

	return len(body), nil
}

func Serve(w io.Writer, h Handler, req *Request, keepAlive bool) error {
	rw := &response{status: 200}
	h.ServeHTTP(rw, req)
	return WriteResponse(w, rw.status, StatusText(rw.status), rw.body, keepAlive, req.Version)
}
