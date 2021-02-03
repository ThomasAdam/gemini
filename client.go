package gemini

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"
)

func defaultCheckRedirect(req *Request, via []*Request) error {
	// The best practices doc
	// (gemini://gemini.circumlunar.space/docs/best-practices.gmi) recommends a
	// maximum of 5 redirects.
	if len(via) >= 5 {
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
	// stop after 5 consecutive requests.
	CheckRedirect func(req *Request, via []*Request) error

	// Identity is the client's identity certificate. It will be sent to the
	// server to authenticate.
	Identity *tls.Certificate
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
	return c.DoContext(context.Background(), req)
}

// DoContext sends a Gemini request and returns a Gemini response, following
// policy (such as redirects, auth) as configured on the client.
//
// The context is only used up to the response status. The response body needs
// to be handled separately.
func (c *Client) DoContext(ctx context.Context, r *Request) (*Response, error) {
	var reqs []*Request

	for {
		resp, err := c.doRequest(ctx, r)
		if err != nil {
			return nil, err
		}

		if resp.statusIsUnknown() {
			return resp, ErrUnknownStatus
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

		// If this isn't a gemini URL, return the raw resp.
		if r.URL.Scheme != "gemini" {
			return resp, ErrUnknownProtocol
		}

		// We need to check redirect policy. If no error is returned, we can try
		// this request as well.
		err = c.checkRedirect(r, reqs)
		if err != nil {
			return resp, err
		}
	}
}

func (c *Client) doRequest(ctx context.Context, r *Request) (*Response, error) {
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

	if c.Identity != nil {
		dialer.Config.Certificates = []tls.Certificate{*c.Identity}
	}

	rawConn, err := dialer.DialContext(ctx, "tcp", hostname+":"+port)
	if err != nil {
		return nil, err
	}
	conn := rawConn.(*tls.Conn)

	type retVal struct {
		resp *Response
		err  error
	}

	// When this function returns, we close this channel, causing some cleanup
	// if needed.
	var doneChan = make(chan struct{})
	defer close(doneChan)

	// This is buffered, but first past the post wins - it's possible for a
	// successful request and a timeout to occur concurrently, so we need to
	// pick one and move on.
	var retChan = make(chan retVal, 2)

	go func() {
		// Wait for either the context or the request to be done. We can't just
		// wait on ctx.Done() because that would cause a goroutine leak whenever
		// context.Background() is used.
		select {
		case <-ctx.Done():
			retChan <- retVal{nil, errors.New("request timed out")}
		case <-doneChan:
		}

	}()

	go func() {
		_, err := conn.Write([]byte(r.String()))
		if err != nil {
			retChan <- retVal{nil, err}
			return
		}

		// The transaction is done, so for good measure, we close our writing side
		// of the connection.
		//
		// NOTE: this seems to break for a number of servers, so it's commented out
		// for now.

		/*
			err = conn.CloseWrite()
			if err != nil {
				return nil, err
			}
		*/

		resp, err := ReadResponse(conn)
		retChan <- retVal{resp, err}
	}()

	ret := <-retChan

	// If this was a failed request, we need to close the connection early to
	// prevent leaking the reader goroutine.
	if ret.resp == nil {
		// Yes, an error is being ignored here, but it's by design.
		_ = rawConn.Close()
	}

	return ret.resp, ret.err
}
