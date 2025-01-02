package vovere

import (
	"encoding/json"
	"io"
	"time"
)

type Metadata struct {
	IndexedAt time.Time `json:"indexed_at"`
}

func (m Metadata) serialize(w io.Writer) error {
	return json.NewEncoder(w).Encode(m)
}
