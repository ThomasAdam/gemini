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

// Client is a Gemini client. Its zero value (DefaultClient) is a usable client
// that uses DefaultTransport.
//
// Clients are safe for concurrent use by multiple goroutines.
type Client struct {
	// CheckRedirect specifies the policy for handling redirects. If
	// CheckRedirect is not nil, the client calls it before following a Gemini
	// redirect. The arguments req and via are the upcoming request and the
	// requests made already, oldest first. If CheckRedirect returns an error,
	// the Client's Get method returns both the previous Response (with its Body
	// closed) and CheckRedirect's error (wrapped in a url.Error) instead of
	// issuing the Request req.
	//
	// If CheckRedirect is nil, the Client uses its default policy, which is to
	// stop after 10 consecutive requests.
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

// Do sends a Gemini request and returns a Gemini response, following policy
// (such as redirects, auth) as configured on the client.
//
// This currently uses context.Background and has no other timeouts.
func (c *Client) Do(req *Request) (*Response, error) {
	// See doContext for more details.
	return c.doContext(context.Background(), req)
}

// DoContext sends a Gemini request and returns a Gemini response, following
// policy (such as redirects, auth) as configured on the client.
func (c *Client) doContext(ctx context.Context, r *Request) (*Response, error) {
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

		// Close the body because we're done with it, otherwise these might end
		// up leaking. Thankfully, there is no connection keepalive, so we can
		// safely close it.
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

		// We need to check redirect policy. If no error is returned, we can try
		// this request as well.
		err = c.checkRedirect(r, reqs)
		if err != nil {
			return resp, err
		}
	}
}

func (c *Client) do(ctx context.Context, r *Request) (*Response, error) {
	hostname := r.URL.Hostname()
	port := r.URL.Port()
	if port == "" {
		port = "1965"
	}

	// TODO: this needs to properly use ctx during reads. Unfortunately I don't
	// know if this is possible, so DoContext is currently private.

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
