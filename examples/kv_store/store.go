package main

import "context"

// KVStore is the interface shimmy will wrap.
type KVStore interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, value string)
}
