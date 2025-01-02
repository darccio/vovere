package vovere

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryPath(t *testing.T) {
	r := Repository{Root: "/path/to/root"}
	testCases := []struct {
		name       string
		uri        string
		collection string
		want       string
	}{
		{
			name: "no collection",
			uri:  "https://example.com/path/to/item",
			want: "/path/to/root/example.com/path/to/item",
		},
		{
			name:       "with collection",
			uri:        "https://example.com/path/to/item",
			collection: "Bookmarks",
			want:       "/path/to/root/Bookmarks/example.com/path/to/item",
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			i := &Item{
				Collection: tc.collection,
			}
			i.URI, _ = ParseURL(tc.uri)
			assert.Equal(t, tc.want, r.Path(i))
		})
	}
}

func TestRepositoryStore(t *testing.T) {
	repo := Repository{
		Root: t.TempDir(),
	}
	var (
		i = &Item{
			Collection: "Bookmarks",
		}
		r = strings.NewReader("testing")
	)
	i.URI, _ = ParseURL("https://example.com/path/to/item")

	// Store file
	err := repo.Store(i, "test.txt", File{Reader: r})
	require.NoError(t, err)

	// Check file copy
	got, err := os.ReadFile(repo.Path(i, "test.txt"))
	require.NoError(t, err)
	require.Contains(t, "testing", string(got))

	// Check metadata
	got, err = os.ReadFile(repo.metadataPath(i, "metadata.json"))
	require.NoError(t, err)
	md := Metadata{}
	err = json.NewDecoder(bytes.NewBuffer(got)).Decode(&md)
	require.NoError(t, err)
	require.NotZero(t, md.IndexedAt)
}
