package gemini

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

type Handler interface {
	Handle(*Request) (*Response, error)
}

type HandlerFunc func(*Request) (*Response, error)

func (hf HandlerFunc) Handle(r *Request) (*Response, error) {
	return hf(r)
}

type Server struct {
	Addr string

	Handler Handler

	TLS *tls.Config
}

func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":1965"
	}

	l, err := tls.Listen("tcp", addr, s.TLS)
	if err != nil {
		return err
	}
	defer l.Close()

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

		rwc := rw.(*tls.Conn)

		go s.serve(rwc)
	}
}

func (s *Server) serve(rwc *tls.Conn) {
	defer rwc.Close()

	req, err := ReadRequest(rwc)
	if err != nil {
		fmt.Println(err)
		return
	}

	resp, err := s.Handler.Handle(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = resp.WriteTo(rwc)
	if err != nil {
		fmt.Println(err)
		return
	}
}
