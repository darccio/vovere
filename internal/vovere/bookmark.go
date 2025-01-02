package vovere

import (
	"encoding/json"
	"io"
	"net/url"
)

type Bookmark struct {
	URI   *url.URL `json:"uri"`
	Title string   `json:"title"`
}

func (b Bookmark) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		URI   string `json:"uri"`
		Title string `json:"title"`
	}{
		URI:   b.URI.String(),
		Title: b.Title,
	})
}

func (b Bookmark) serialize(w io.Writer) error {
	return json.NewEncoder(w).Encode(b)
}
