package model

import "time"

// EvictionPolicy determines how entries are removed when capacity is reached.
type EvictionPolicy string

const (
	PolicyLRU  EvictionPolicy = "lru"
	PolicyLFU  EvictionPolicy = "lfu"
	PolicyFIFO EvictionPolicy = "fifo"
)

// Config holds cache configuration.
type Config struct {
	MaxEntries     int
	MaxMemoryBytes int64
	EvictionPolicy EvictionPolicy
	ShardCount     int
	VirtualNodes   int
	WALEnabled     bool
	WALPath        string
	ListenAddr     string
	DefaultTTL     time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxEntries:     100000,
		MaxMemoryBytes: 256 * 1024 * 1024, // 256MB
		EvictionPolicy: PolicyLRU,
		ShardCount:     64,
		VirtualNodes:   128,
		WALEnabled:     false,
		WALPath:        "kvcache.wal",
		ListenAddr:     ":8080",
		DefaultTTL:     0,
	}
}

// Entry represents a stored cache item.
type Entry struct {
	Key       string    `json:"key"`
	Value     []byte    `json:"value"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired reports whether the entry has passed its TTL.
func (e *Entry) IsExpired(now time.Time) bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return now.After(e.ExpiresAt)
}
