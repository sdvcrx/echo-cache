package cache

import (
	"net/http"
	"time"

	"encoding/json"
)

type Response struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
}

func (r *Response) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Response) Unmarshal(b []byte) error {
	return json.Unmarshal(b, r)
}

func NewResponse(code int, header http.Header, body []byte) *Response {
	return &Response{
		StatusCode: code,
		Headers:    header,
		Body:       body,
	}
}

func NewResponseFromJSON(s string) (*Response, error) {
	r := Response{}
	err := r.Unmarshal([]byte(s))
	return &r, err
}

type CacheAdapter interface {
	Get(key string) (*Response, error)
	Set(key string, response *Response, ttl time.Duration) error
}
