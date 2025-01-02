package vovere

import "io"

type Blob interface {
	// Supported internal types implement this method to keep the interface closed.
	serialize(w io.Writer) error
}
