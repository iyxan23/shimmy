package main

import "context"

// MapStore is a basic in-memory store.
type MapStore struct {
	data map[string]string
}

func NewMapStore() *MapStore {
	return &MapStore{data: map[string]string{}}
}

func (s *MapStore) Get(_ context.Context, key string) (string, bool) {
	v, ok := s.data[key]
	return v, ok
}

func (s *MapStore) Set(_ context.Context, key string, value string) {
	s.data[key] = value
}
