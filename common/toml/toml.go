package toml

import (
	"io"

	"github.com/BurntSushi/toml"
)

// import "github.com/naoina/toml"

func Marshal(v any) ([]byte, error) {
	return toml.Marshal(v)
}

func Unmarshal(data []byte, ptr any) error {
	return toml.Unmarshal(data, ptr)
}

func NewDecoder(r io.Reader) *toml.Decoder {
	return toml.NewDecoder(r)
}

func NewEncoder(w io.Writer) *toml.Encoder {
	return toml.NewEncoder(w)
}
