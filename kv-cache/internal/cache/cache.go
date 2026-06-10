package cache

import (
	"context"
	"time"

	"kv-cache/internal/model"
)

// Cache is the top-level key-value cache interface.
// All methods are safe for concurrent use.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Stats(ctx context.Context) model.Stats
	Close() error
}

// Clock abstracts time for deterministic testing.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
