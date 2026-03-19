package main

import (
	"context"
	"fmt"
	"time"

	"shimmy"
)

// UserServiceLoggingMiddleware logs typed method calls.
type UserServiceLoggingMiddleware struct {
	BaseUserServiceMiddleware
}

func (m *UserServiceLoggingMiddleware) AroundCreateUser(
	ctx context.Context,
	id string,
	name string,
	next func(context.Context, string, string) (User, error),
) (User, error) {
	fmt.Printf("[typed-mw] CreateUser id=%s name=%s\n", id, name)
	return next(ctx, id, name)
}

// NewLatencyInterceptor reports method latency across all methods.
func NewLatencyInterceptor() UserServiceMiddleware {
	return NewUserServiceInterceptor(func(call shimmy.Call, invoke func()) {
		start := time.Now()
		invoke()
		fmt.Printf("[interceptor] %s took %s\n", call.Method(), time.Since(start))
	})
}
