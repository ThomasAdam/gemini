package gemini

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

func defaultCheckRedirect(req *Request, via []*Request) error {
	if len(via) >= 10 {
		return errors.New("too many redirects")
	}

	return nil
}

type Client struct {
	CheckRedirect func(req *Request, via []*Request) error
}

// checkRedirect calls either the user's configured CheckRedirect function, or
// the default.
func (c *Client) checkRedirect(req *Request, via []*Request) error {
	fn := c.CheckRedirect
	if fn == nil {
		fn = defaultCheckRedirect
	}
	return fn(req, via)
}

func (c *Client) Do(r *Request) (*Response, error) {
	return c.DoContext(context.Background(), r)
}

func (c *Client) DoContext(ctx context.Context, r *Request) (*Response, error) {
	var reqs []*Request

	for {
		resp, err := c.do(ctx, r)
		if err != nil {
			return nil, err
		}

		// If it wasn't a redirect, this request is done.
		if !resp.IsRedirect() {
			return resp, nil
		}

		// Close the body because we're done with it.
		err = resp.Body.Close()
		if err != nil {
			return nil, err
		}

		// Add the current request to the request chain before making a new
		// request.
		reqs = append(reqs, r)
		ref, err := url.Parse(resp.Meta)
		if err != nil {
			return nil, err
		}
		r = NewRequestURL(r.URL.ResolveReference(ref))

		err = c.checkRedirect(r, reqs)
		if err != nil {
			return nil, err
		}
	}
}

func (c *Client) do(ctx context.Context, r *Request) (*Response, error) {
	hostname := r.URL.Hostname()
	port := r.URL.Port()
	if port == "" {
		port = "1965"
	}

	// TODO: this needs to be better. Unfortunately the spec allows/recommends
	// that people not set up letsencrypt or something similar, so we will need
	// to handle that another way. The generally accepted method is TOFU (trust
	// on first use).
	dialer := &tls.Dialer{
		Config: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
	}
	rawConn, err := dialer.DialContext(ctx, "tcp", hostname+":"+port)
	if err != nil {
		return nil, err
	}
	conn := rawConn.(*tls.Conn)

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(r.URL.String())
	if err != nil {
		return nil, err
	}

	_, err = writer.WriteString("\r\n")
	if err != nil {
		return nil, err
	}

	err = writer.Flush()
	if err != nil {
		return nil, err
	}

	// The transaction is done, so for good measure, we close our writing side
	// of the connection. NOTE: this seems to break for a number of servers, so
	// it's commented out for now.
	/*
		err = conn.CloseWrite()
		if err != nil {
			return nil, err
		}
	*/

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

	split := strings.SplitN(line, " ", 2)
	if len(split) != 2 {
		return nil, errors.New("invalid response")
	}

	status, err := strconv.Atoi(split[0])
	if err != nil {
		return nil, err
	}

	return &Response{
		Status: status,
		Meta:   split[1],
		Body: &wrappedBufferedReader{
			buf: reader,
			rc:  conn,
		},
	}, nil
}
