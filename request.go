package gemini

import (
	"net/url"
)

// A Request represents a Gemini request received by a server or to be sent by a
// client.
type Request struct {
	URL *url.URL
}

func (r *Request) String() string {
	return r.URL.String() + "\r\n"
}

// NewRequest returns a new Request given a URL in string form.
func NewRequest(rawUrl string) (*Request, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	return &Request{
		URL: url,
	}, nil
}

// NewRequestURL returns a new Request given a URL.
func NewRequestURL(url *url.URL) *Request {
	return &Request{
		URL: url,
	}
}
