package vovere

import (
	"io"
)

type File struct {
	Reader io.Reader
}

func (f File) serialize(w io.Writer) error {
	_, err := io.Copy(w, f.Reader)
	return err
}
