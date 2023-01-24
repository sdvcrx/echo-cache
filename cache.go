package cache

import (
	"net/http"
)

type Response struct {
	StatusCode int         `msgpack:"status_code"`
	Headers    http.Header `msgpack:"headers,omitempty"`
	Body       []byte      `msgpack:"body,omitempty"`
}

func NewResponse(code int, header http.Header, body []byte) *Response {
	return &Response{
		StatusCode: code,
		Headers:    header,
		Body:       body,
	}
}
