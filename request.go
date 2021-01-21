package gemini

import (
	"net/url"
)

type Request struct {
	URL *url.URL
}

func NewRequest(rawUrl string) (*Request, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	return &Request{
		URL: url,
	}, nil
}

func NewRequestURL(url *url.URL) *Request {
	return &Request{
		URL: url,
	}
}
