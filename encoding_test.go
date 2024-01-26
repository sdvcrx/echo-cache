package cache

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoder(t *testing.T) {
	encs := map[string]Encoder{
		"JSON":    &JSONEncoder{},
		"Msgpack": &MsgpackEncoder{},
	}

	resp := NewResponse(
		http.StatusOK,
		http.Header{
			"User-Agent": []string{"curl"},
		},
		[]byte("OK"),
	)

	for name, enc := range encs {
		t.Run(name+" Encoder", func(t *testing.T) {
			data, err := enc.Marshal(resp)
			assert.NoError(t, err)

			newResp := Response{}
			err = enc.Unmarshal(data, &newResp)
			assert.NoError(t, err)
			assert.EqualValues(t, *resp, newResp)
		})
	}
}

func BenchmarkEncoderMarshal(b *testing.B) {
	encoders := map[string]Encoder{
		"JSON":    &JSONEncoder{},
		"Msgpack": &MsgpackEncoder{},
	}

	response := &Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"key": "value"}`),
	}

	for name, encoder := range encoders {
		data, _ := encoder.Marshal(response)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(len(data)))

			for i := 0; i < b.N; i++ {
				_, err := encoder.Marshal(response)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEncoderUnmarshal(b *testing.B) {
	encoders := map[string]Encoder{
		"JSON":    &JSONEncoder{},
		"Msgpack": &MsgpackEncoder{},
	}

	response := &Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"application/json"}},
		Body:       []byte(`{"key": "value"}`),
	}

	for name, encoder := range encoders {
		data, err := encoder.Marshal(response)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.SetBytes(int64(len(data)))

			for i := 0; i < b.N; i++ {
				err := encoder.Unmarshal(data, response)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
