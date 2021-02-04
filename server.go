package gemini

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/url"
	"runtime"
	"strings"
	"time"
)

// A ResponseWriter interface is used by a Gemini handler to construct a Gemini
// response.
//
// A ResponseWriter may not be used after the Handler.ServeGemini method has
// returned.
type ResponseWriter interface {
	// Write writes the data to the connection as part of a Gemini reply.
	//
	// If WriteStatus has not yet been called, Write calls
	// WriteStatus(gemini.StatusSuccess, "text/gemini") before writing the data.
	Write([]byte) (int, error)

	// WriteStatus sends a Gemini status response with the provided status code
	// and meta text.
	//
	// If WriteStatus is not called explicitly, the first call to Write will
	// trigger an implicit WriteStatus(gemini.StatusSuccess, "text/gemini").
	// Explicit calls to WriteStatus are generally used to specify error codes
	// or alternate content types.
	//
	// The provided code must be a valid Gemini status code.
	WriteStatus(statusCode int, meta string)
}

// Params is a convenience wrapper around []string, used for storing URL params.
type Params []string

// A Handler responds to a Gemini request.
//
// If ServeGemini panics, the server (the caller of ServeGemini) assumes that
// the effect of the panic was isolated to the active request. It recovers the
// panic, logs a stack trace to the server error log, and closes the network
// connection. To abort a handler so the client sees an interrupted response but
// the server doesn't log an error, panic with the value ErrAbortHandler.
type Handler interface {
	ServeGemini(context.Context, *Request, ResponseWriter)
}

// HandlerFunc adapts a function to work as a full Handler.
type HandlerFunc func(context.Context, *Request, ResponseWriter)

func (hf HandlerFunc) ServeGemini(ctx context.Context, r *Request, w ResponseWriter) {
	hf(ctx, r, w)
}

type Server struct {
	Addr    string
	Handler Handler
	TLS     *tls.Config
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
	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Printf("gemini: panic serving %v: %v\n%s", rwc.RemoteAddr(), err, buf)
		}
	}()

	defer rwc.Close()

	req, err := ReadRequest(rwc)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("--> %s\n", req.URL)

	writer := newResponseWriter(rwc)

	s.Handler.ServeGemini(context.TODO(), req, writer)

	if !writer.hasWritten {
		writer.WriteStatus(StatusNotFound, "not found")
	}

	fmt.Printf("<-- %d %s\n", writer.writtenStatus, writer.writtenMeta)
}

func StripPrefix(prefix string, h Handler) Handler {
	if prefix == "" {
		return h
	}

	return HandlerFunc(func(ctx context.Context, r *Request, w ResponseWriter) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p

			h.ServeGemini(ctx, r2, w)
		}
	})
}

type responseWriter struct {
	writtenStatus int
	writtenMeta   string
	hasWritten    bool

	w io.Writer
}

func newResponseWriter(w io.Writer) *responseWriter {
	return &responseWriter{w: w}
}

func (w *responseWriter) Write(data []byte) (int, error) {
	if !w.hasWritten {
		w.WriteStatus(StatusSuccess, "text/gemini")
	}

	return w.w.Write(data)
}

func (w *responseWriter) WriteStatus(statusCode int, meta string) {
	if w.hasWritten {
		fmt.Println("Cannot write status multiple times")
		return
	}

	w.writtenStatus = statusCode
	w.writtenMeta = meta
	w.hasWritten = true

	fmt.Fprintf(w.w, "%d %s\r\n", statusCode, meta)
}
