package cache

import (
	"net/http"
	"time"

	"encoding/json"
)

type Response struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers,omitempty"`
	Body       []byte      `json:"body,omitempty"`
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
