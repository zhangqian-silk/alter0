package domain

import "context"

type LiveUserMessageSource interface {
	ConsumeLatest(ctx context.Context) (string, bool)
}

type liveUserMessageContextKey struct{}

func WithLiveUserMessageSource(ctx context.Context, source LiveUserMessageSource) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if source == nil {
		return ctx
	}
	return context.WithValue(ctx, liveUserMessageContextKey{}, source)
}

func ConsumeLiveUserMessage(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	source, ok := ctx.Value(liveUserMessageContextKey{}).(LiveUserMessageSource)
	if !ok || source == nil {
		return "", false
	}
	return source.ConsumeLatest(ctx)
}
