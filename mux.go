package gemini

import "context"

type ServeMux struct {
	root *node
}

func NewServeMux() *ServeMux {
	return &ServeMux{
		root: newNode(),
	}
}

func (mux *ServeMux) ServeGemini(ctx context.Context, r *Request) *Response {
	return mux.root.ServeGemini(ctx, r)
}

func (mux *ServeMux) Handle(pattern string, h Handler) {
	mux.root.Handle(pattern, h)
}

func (mux *ServeMux) NotFound(h Handler) {
	mux.root.NotFound(h)
}

func (mux *ServeMux) Route(pattern string, fn func(r Router)) Router {
	return mux.root.Route(pattern, fn)
}

type Router interface {
	Handler
	Handle(pattern string, h Handler)
	NotFound(h Handler)
	Route(pattern string, fn func(r Router)) Router
}
