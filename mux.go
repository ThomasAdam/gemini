package gemini

import "context"

type ServeMux struct {
	RedirectSlash bool
	root          *node
}

func NewServeMux() *ServeMux {
	return &ServeMux{
		RedirectSlash: true,
		root:          newNode(),
	}
}

func (mux *ServeMux) ServeGemini(ctx context.Context, r *Request, w ResponseWriter) {
	params, handler := mux.root.match(r.URL.Path, mux.RedirectSlash)
	if handler == nil {
		return
	}

	ctx = CtxWithParams(ctx, params)
	handler.ServeGemini(ctx, r, w)
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
	Handle(pattern string, h Handler)
	NotFound(h Handler)
	Route(pattern string, fn func(r Router)) Router
}
