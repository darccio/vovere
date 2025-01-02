package vovere

import (
	"errors"
	"net/url"
)

func ParseURL(rawurl string) (*url.URL, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, errors.New("invalid url")
	}
	return u, nil
}
