package vovere

import (
	"os"
	"path/filepath"
	"time"
)

// Repository represents a repository of items.
type Repository struct {
	// Root is the root directory of the repository.
	Root string
}

// Path returns the absolute path to the item's subpath in the repository.
// No subpath provided returns the item's root directory.
func (r Repository) Path(i *Item, subpathParts ...string) string {
	return filepath.Join(
		r.Root,
		i.Collection,
		i.URI.Hostname(),
		i.URI.Path,
		filepath.Join(subpathParts...),
	)
}

// Store stores an item in the repository.
func (r Repository) Store(i *Item, fname string, b Blob) error {
	md := Metadata{
		IndexedAt: time.Now(),
	}
	if err := r.storeMetadata(i, "metadata.json", md); err != nil {
		return err
	}
	return r.store(i, fname, b, r.Path)
}

func (r Repository) storeMetadata(i *Item, fname string, b Blob) error {
	return r.store(i, fname, b, r.metadataPath)
}

// metadataPath returns the absolute path to the item's metadata subpath in the repository.
// No subpath provided returns the item's metadata directory.
func (r Repository) metadataPath(i *Item, subpathParts ...string) string {
	return r.Path(i, ".vovere", filepath.Join(subpathParts...))
}

type pathResolver = func(*Item, ...string) string

func (r *Repository) store(i *Item, fname string, b Blob, pr pathResolver) error {
	mdPath := pr(i)
	if err := os.MkdirAll(mdPath, 0755); err != nil {
		return err
	}
	path := pr(i, fname)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return b.serialize(f)
}
