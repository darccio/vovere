package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/vovere/internal/vovere"
	"github.com/adrg/xdg"
	"github.com/go-rod/rod"
)

const (
	scheme = "vovere"
)

var (
	errNoURL          = errors.New("URL is required")
	errSchemeRequired = fmt.Errorf("URL must be a %q URL", scheme)
	errUnknownCommand = errors.New("unknown command")
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "add":
		var (
			url         *url.URL
			defaultRepo *vovere.Repository
		)
		url, err = vovere.ParseURL(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}

		defaultRepo = &vovere.Repository{
			Root: xdg.UserDirs.Documents,
		}
		if len(os.Args) < 4 {
			err = add(defaultRepo, url, "")
			break
		}
		err = add(defaultRepo, url, os.Args[3])
	// case "tag"
	// case "link"
	// case "pop" // pop random item, ask to discard or keep; if kept, add to counter
	// case "archive"
	// case "note"
	default:
		err = errUnknownCommand
	}
	if err != nil {
		log.Fatal(err)
	}
}

func add(repo *vovere.Repository, url *url.URL, fpath string) error {
	if url == nil {
		return errNoURL
	}
	fpath = strings.TrimSpace(fpath)
	if fpath == "" {
		return addBookmark(repo, url)
	}
	return addPath(repo, url, filepath.Clean(fpath))
}

func addBookmark(repo *vovere.Repository, url *url.URL) error {
	i := &vovere.Item{
		URI:        url,
		Collection: "Bookmarks",
	}
	title, err := getTitle(url)
	if err != nil {
		return err
	}
	bm := &vovere.Bookmark{
		URI:   url,
		Title: title,
	}
	return repo.Store(i, "bookmark.json", bm)
}

func getTitle(url *url.URL) (string, error) {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(url.String()).MustWaitStable()
	return page.MustElement("title").Text()
}

func addPath(repo *vovere.Repository, url *url.URL, fpath string) error {
	if url.Scheme != scheme {
		return errSchemeRequired
	}
	i := &vovere.Item{
		URI: url,
	}
	r, err := os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer r.Close()
	fname := filepath.Base(fpath)
	f := &vovere.File{
		Reader: r,
	}
	return repo.Store(i, fname, f)
}
