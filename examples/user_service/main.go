package main

import (
	"context"
	"fmt"
)

func main() {
	inner := NewInMemoryUserService()

	service := &UserServiceShim{
		Inner: inner,
		Middleware: []UserServiceMiddleware{
			&UserServiceLoggingMiddleware{},
			NewLatencyInterceptor(),
		},
	}

	ctx := context.Background()

	created, err := service.CreateUser(ctx, "u-1", "Ari")
	if err != nil {
		fmt.Println("create error:", err)
		return
	}
	fmt.Printf("created user: %+v\n", created)

	fetched, err := service.GetUser(ctx, "u-1")
	if err != nil {
		fmt.Println("get error:", err)
		return
	}
	fmt.Printf("fetched user: %+v\n", fetched)
}
