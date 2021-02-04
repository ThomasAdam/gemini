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

// NotFound replies to the request with a gemini.StatusNotFound error.
func NotFound(ctx context.Context, r *Request, w ResponseWriter) {
	w.WriteStatus(StatusNotFound, "not found")
}

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
	ServeGemini(context.Context, ResponseWriter, *Request)
}

// HandlerFunc adapts a function to work as a full Handler.
type HandlerFunc func(context.Context, ResponseWriter, *Request)

func (hf HandlerFunc) ServeGemini(ctx context.Context, w ResponseWriter, r *Request) {
	hf(ctx, w, r)
}

// A Server defines parameters for running a Gemini server. The zero value for
// Server is a valid configuration, though it won't do very much.
//
// Before usage, the TLS config must either have a valid certificate or a
// GetCertificate callback.
//
// If you wish to get client certificates, you must set ClientAuth in the TLS
// config to at least RequestClientCert.
type Server struct {
	Addr    string
	Handler Handler
	TLS     *tls.Config
}

// Serve accepts incoming connections on the Listener l, creating a new service
// goroutine for each. The service goroutines read requests and then call
// srv.Handler to reply to them.
//
// Serve always returns a non-nil error and closes l.
func (s *Server) Serve(l net.Listener) error {
	defer l.Close()

	tlsConfig := s.TLS.Clone()

	// If the MinVersion has not been set, set it to what the spec recommends.
	if tlsConfig.MinVersion == 0 {
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		conn, err := l.Accept()
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

		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
		}

		rwc := tls.Server(conn, tlsConfig)
		go s.serve(rwc)
	}
}

// ListenAndServe listens on the TCP network address srv.Addr and then calls
// Serve to handle requests on incoming connections. Accepted connections are
// configured to enable TCP keep-alives.
//
// If srv.Addr is blank, ":1965" is used.
//
// ListenAndServe always returns a non-nil error. After Shutdown or Close, the
// returned error is ErrServerClosed.
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
	writer := newResponseWriter(rwc)

	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Printf("gemini: panic serving %v: %v\n%s", rwc.RemoteAddr(), err, buf)
		}

		if !writer.hasWritten {
			writer.WriteStatus(StatusCGIError, "internal panic")
		}
	}()

	defer rwc.Close()

	req, err := ReadRequest(rwc)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("--> %s\n", req.URL)

	if s.Handler != nil {
		s.Handler.ServeGemini(context.TODO(), writer, req)
	}

	if !writer.hasWritten {
		NotFound(context.TODO(), req, writer)
	}

	fmt.Printf("<-- %d %s\n", writer.writtenStatus, writer.writtenMeta)
}

// StripPrefix returns a handler that serves requests by removing the given
// prefix from the request URL's Path and invoking the handler h. StripPrefix
// handles a request for a path that doesn't begin with prefix by letting it
// fall through to another handler, generally returning gemini.StatusNotFound.
func StripPrefix(prefix string, h Handler) Handler {
	if prefix == "" {
		return h
	}

	return HandlerFunc(func(ctx context.Context, w ResponseWriter, r *Request) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := new(Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p

			h.ServeGemini(ctx, w, r2)
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
