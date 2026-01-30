package store

import "context"

// Store - represents a generic key-value store interface
type Store interface {
	Get(ctx context.Context, key string) (UrlObject, error)
	Put(ctx context.Context, key string, value string) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]UrlObject, error)
}

type UrlObject struct {
	Key           string
	URL           string
	RedirectCount int
}
