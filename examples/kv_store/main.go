package main

import (
	"context"
	"fmt"
)

func main() {
	inner := NewMapStore()

	store := &KVStoreShim{
		Inner: inner,
		Middleware: []KVStoreMiddleware{
			&CacheStatsMiddleware{},
			NewAuditInterceptor(),
		},
	}

	ctx := context.Background()

	store.Set(ctx, "session", "abc123")
	v1, ok1 := store.Get(ctx, "session")
	fmt.Printf("get existing: value=%s ok=%v\n", v1, ok1)

	v2, ok2 := store.Get(ctx, "missing")
	fmt.Printf("get missing: value=%q ok=%v\n", v2, ok2)
}
