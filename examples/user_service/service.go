package main

import "context"

// User is a small DTO used by UserService.
type User struct {
	ID   string
	Name string
}

// UserService is the interface shimmy will wrap.
type UserService interface {
	GetUser(ctx context.Context, id string) (User, error)
	CreateUser(ctx context.Context, id string, name string) (User, error)
}
