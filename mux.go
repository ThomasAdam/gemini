package gemini

import "context"

// Mux is a simple Gemini route multiplexer that parses a request path, records
// any URL params, and executes an end handler. It implements the gemini.Handler
// interface.
//
// Mux is designed to be fast, minimal and offer a powerful API for building
// modular and composable HTTP services with a large set of handlers. It's
// particularly useful for writing large REST API services that break a handler
// into many smaller parts composed of middlewares and end handlers.
type ServeMux struct {
	RedirectSlash bool
	root          *node
}

// NewServeMux returns a newly initialized ServeMux object that implements the
// Router interface.
func NewServeMux() *ServeMux {
	s := &ServeMux{
		RedirectSlash: true,
	}

	root := newNode(nil)
	root.mux = s
	s.root = root

	return s
}

// ServeGemini implements the gemini.Handler interface.
func (mux *ServeMux) ServeGemini(ctx context.Context, w ResponseWriter, r *Request) {
	mux.root.ServeGemini(ctx, w, r)
}

// Handle adds the route `pattern` to execute the `handler` gemini.Handler.
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.root.Handle(pattern, handler)
}

// NotFound sets a custom gemini.Handler for routing paths that could not be
// found. The default 404 handler is `gemini.NotFound`.
func (mux *ServeMux) NotFound(handler Handler) {
	mux.root.NotFound(handler)
}

// Route effectively defines a new subrouter, mounted to `pattern`.
func (mux *ServeMux) Route(pattern string, fn func(r Router)) Router {
	return mux.root.Route(pattern, fn)
}

// Router consisting of the core routing methods used by ServeMux.
type Router interface {
	Handler
	Handle(pattern string, h Handler)
	NotFound(h Handler)
	Route(pattern string, fn func(r Router)) Router
}
