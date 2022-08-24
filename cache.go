package cache

import (
	"net/http"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type Response struct {
	StatusCode int         `msgpack:"status_code"`
	Headers    http.Header `msgpack:"headers,omitempty"`
	Body       []byte      `msgpack:"body,omitempty"`
}

func (r *Response) Marshal() ([]byte, error) {
	return msgpack.Marshal(r)
}

func (r *Response) Unmarshal(b []byte) error {
	return msgpack.Unmarshal(b, r)
}

func NewResponse(code int, header http.Header, body []byte) *Response {
	return &Response{
		StatusCode: code,
		Headers:    header,
		Body:       body,
	}
}

func NewResponseFromJSON(jsonb []byte) (*Response, error) {
	r := Response{}
	err := r.Unmarshal(jsonb)
	return &r, err
}

func NewResponseFromJSONString(s string) (*Response, error) {
	return NewResponseFromJSON([]byte(s))
}

type CacheAdapter interface {
	Get(key string) (*Response, error)
	Set(key string, response *Response, ttl time.Duration) error
}
