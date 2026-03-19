package main

import (
	"context"
	"fmt"

	"shimmy"
)

// CacheStatsMiddleware prints hit/miss information for Get calls.
type CacheStatsMiddleware struct {
	BaseKVStoreMiddleware
}

func (m *CacheStatsMiddleware) AroundGet(
	ctx context.Context,
	key string,
	next func(context.Context, string) (string, bool),
) (string, bool) {
	v, ok := next(ctx, key)
	if ok {
		fmt.Printf("[typed-mw] cache hit key=%s value=%s\n", key, v)
	} else {
		fmt.Printf("[typed-mw] cache miss key=%s\n", key)
	}
	return v, ok
}

// NewAuditInterceptor prints generic method metadata.
func NewAuditInterceptor() KVStoreMiddleware {
	return NewKVStoreInterceptor(func(call shimmy.Call, invoke func()) {
		fmt.Printf("[interceptor] calling %s with %d args\n", call.Method(), len(call.Args()))
		invoke()
	})
}
