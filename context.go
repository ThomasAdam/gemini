package gemini

import "context"

type contextKey string

const (
	ctxKeyParams contextKey = "params"
)

// CtxWithParams overwrites the params stored in the request context. This is
// generally only useful for internal code and middleware.
func CtxWithParams(ctx context.Context, params Params) context.Context {
	return context.WithValue(ctx, ctxKeyParams, params)
}

// CtxParams allows you to extract the URL params from a request context.
func CtxParams(ctx context.Context) Params {
	val := ctx.Value(ctxKeyParams)
	if val == nil {
		return Params{}
	}

	return val.(Params)
}
