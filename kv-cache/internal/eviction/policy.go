package eviction

import "kv-cache/internal/model"

// Policy tracks access patterns and decides which key to evict.
// Implementations are NOT thread-safe; the caller must hold a lock.
type Policy interface {
	// Access records that a key was accessed.
	Access(key string)

	// Add records that a key was inserted. Returns the evicted key
	// if capacity is reached, or empty string if no eviction needed.
	Add(key string) (evicted string)

	// Remove removes a key from tracking.
	Remove(key string)

	// Len returns the number of tracked keys.
	Len() int
}

// New creates a Policy for the given eviction strategy and capacity.
func New(policy model.EvictionPolicy, capacity int) Policy {
	switch policy {
	case model.PolicyLFU:
		return newLFU(capacity)
	case model.PolicyFIFO:
		return newFIFO(capacity)
	default:
		return newLRU(capacity)
	}
}
