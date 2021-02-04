package gemini

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

type Params map[string]string

type Handler interface {
	ServeGemini(context.Context, *Request) *Response
}

type HandlerFunc func(context.Context, *Request) *Response

func (hf HandlerFunc) ServeGemini(ctx context.Context, r *Request) *Response {
	return hf(ctx, r)
}

type Server struct {
	Addr string

	Handler Handler

	TLS *tls.Config
}

func (s *Server) Serve(l net.Listener) error {
	defer l.Close()

	tlsConfig := s.TLS.Clone()

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, err := l.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				time.Sleep(tempDelay)
				continue
			}

			return err
		}

		rwc := tls.Server(rw, tlsConfig)

		go s.serve(rwc)
	}
}

func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":1965"
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return s.Serve(l)
}

func (s *Server) serve(rwc *tls.Conn) {
	defer rwc.Close()

	req, err := ReadRequest(rwc)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("--> %s\n", req.URL)

	resp := s.Handler.ServeGemini(context.TODO(), req)
	if resp == nil {
		resp = NewResponse(StatusNotFound, "not found")
	}

	fmt.Printf("<-- %d %s\n", resp.Status, resp.Meta)

	_, err = resp.WriteTo(rwc)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func StripPrefix(prefix string, h Handler) Handler {
	if prefix == "" {
		return h
	}

	return HandlerFunc(func(ctx context.Context, r *Request) *Response {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p

			return h.ServeGemini(ctx, r2)
		} else {
			return nil
		}
	})
}
