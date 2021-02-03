package gemini

import "context"

type contextKey string

const (
	ctxKeyParams contextKey = "params"
)

func CtxWithParams(ctx context.Context, params []string) context.Context {
	return context.WithValue(ctx, ctxKeyParams, params)
}

func CtxParams(ctx context.Context) []string {
	val := ctx.Value(ctxKeyParams)
	if val == nil {
		return nil
	}

	return val.([]string)
}
