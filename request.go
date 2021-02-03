package gemini

import (
	"bufio"
	"crypto/tls"
	"errors"
	"io"
	"net/url"
	"strings"
)

// A Request represents a Gemini request received by a server or to be sent by a
// client.
type Request struct {
	// TODO: find a good api for making a request where the connection URL and
	// Request URL are different.
	URL *url.URL

	// TLS allows Gemini servers and other software to record information about
	// the TLS connection on which the request was received.
	TLS *tls.ConnectionState
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

func ReadRequest(conn io.ReadCloser) (*Request, error) {
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// This check needs to be here, otherwise TrimSuffix won't be able to
	// guarantee that we're getting valid lines.
	if !strings.HasSuffix(line, "\r\n") {
		return nil, errors.New("malformed status line")
	}

	line = strings.TrimSuffix(line, "\r\n")

	url, err := url.Parse(line)
	if err != nil {
		return nil, err
	}

	ret := &Request{
		URL: url,
	}

	if tc, ok := conn.(*tls.Conn); ok {
		state := tc.ConnectionState()
		ret.TLS = &state
	}

	return ret, nil
}
