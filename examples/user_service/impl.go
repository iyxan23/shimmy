package main

import (
	"context"
	"fmt"
)

// InMemoryUserService is a user-level concrete implementation.
type InMemoryUserService struct {
	users map[string]User
}

func NewInMemoryUserService() *InMemoryUserService {
	return &InMemoryUserService{users: map[string]User{}}
}

func (s *InMemoryUserService) GetUser(_ context.Context, id string) (User, error) {
	u, ok := s.users[id]
	if !ok {
		return User{}, fmt.Errorf("user %q not found", id)
	}
	return u, nil
}

func (s *InMemoryUserService) CreateUser(_ context.Context, id string, name string) (User, error) {
	u := User{ID: id, Name: name}
	s.users[id] = u
	return u, nil
}
