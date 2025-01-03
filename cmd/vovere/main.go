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
	tea "github.com/charmbracelet/bubbletea"
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
	m := newTUIModel()
	m.handler = func() (string, error) {
		var (
			url *url.URL
			err error
			msg string
		)
		switch os.Args[1] {
		case "add":
			url, err = vovere.ParseURL(os.Args[2])
			if err != nil {
				return "", err
			}
			defaultRepo := &vovere.Repository{
				Root: xdg.UserDirs.Documents,
			}
			var fpath string
			if len(os.Args) >= 4 {
				fpath = os.Args[3]
			}
			msg, err = add(defaultRepo, url, fpath)
		// case "tag"
		// case "link"
		// case "pop" // pop random item, ask to discard or keep; if kept, add to counter
		// case "archive"
		// case "note"
		default:
			err = errUnknownCommand
		}
		return msg, err
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func add(repo *vovere.Repository, url *url.URL, fpath string) (string, error) {
	if url == nil {
		return "", errNoURL
	}
	fpath = strings.TrimSpace(fpath)
	if fpath == "" {
		return addBookmark(repo, url)
	}
	return addPath(repo, url, filepath.Clean(fpath))
}

func addBookmark(repo *vovere.Repository, url *url.URL) (string, error) {
	i := &vovere.Item{
		URI:        url,
		Collection: "Bookmarks",
	}
	title, err := getTitle(url)
	if err != nil {
		return "", err
	}
	bm := &vovere.Bookmark{
		URI:   url,
		Title: title,
	}
	if err := repo.Store(i, "bookmark.json", bm); err != nil {
		return "", err
	}
	return fmt.Sprintf("added bookmark %q", title), nil
}

func getTitle(url *url.URL) (string, error) {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(url.String()).MustWaitLoad()
	return page.MustElement("title").Text()
}

func addPath(repo *vovere.Repository, url *url.URL, fpath string) (string, error) {
	if url.Scheme != scheme {
		return "", errSchemeRequired
	}
	i := &vovere.Item{
		URI: url,
	}
	r, err := os.OpenFile(fpath, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer r.Close()
	fname := filepath.Base(fpath)
	f := &vovere.File{
		Reader: r,
	}
	if err = repo.Store(i, fname, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("added file %q", fpath), nil
}
