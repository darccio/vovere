package vovere

import (
	"net/url"
)

// Item represents an item in a repository.
type Item struct {
	// URI is the URL of the item.
	URI *url.URL

	// Collection is the collection of the item. It groups items under an alternate hierarchy.
	Collection string
}
