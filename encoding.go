package cache

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

type Marshaler interface {
	Marshal(r *Response) ([]byte, error)
}

type Unmarshaler interface {
	Unmarshal(b []byte, v *Response) error
}

// Interface that marshal/unmarshal `Response`
type Encoder interface {
	Marshaler
	Unmarshaler
}

type MsgpackEncoder struct{}

func (m *MsgpackEncoder) Marshal(r *Response) ([]byte, error) {
	return msgpack.Marshal(r)
}

func (m *MsgpackEncoder) Unmarshal(b []byte, v *Response) error {
	return msgpack.Unmarshal(b, v)
}

var _ Encoder = &MsgpackEncoder{}

type JSONEncoder struct{}

func (e *JSONEncoder) Marshal(r *Response) ([]byte, error) {
	return json.Marshal(r)
}

func (e *JSONEncoder) Unmarshal(b []byte, v *Response) error {
	return json.Unmarshal(b, v)
}

var _ Encoder = &JSONEncoder{}
