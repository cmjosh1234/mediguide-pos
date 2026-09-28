package requestctx

import "context"

type correlationIDKey struct{}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationIDKey{}, id)
}

func CorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(correlationIDKey{}).(string)
	return value
}
