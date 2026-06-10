package ratelimit

import (
	"hash/fnv"
	"sync"
	"time"
)

const limiterShards = 256

// Config defines rate limiting parameters.
type Config struct {
	Rate     float64 // tokens per second
	Capacity float64 // max burst size
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type limiterShard struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

// Limiter enforces rate limits using a sharded token bucket algorithm.
// Concurrency model: 256 shards, each with its own Mutex.
// Keys are hashed to determine shard, reducing contention.
type Limiter struct {
	shards   [limiterShards]limiterShard
	rate     float64
	capacity float64
}

// New creates a Limiter with the given rate and capacity.
func New(cfg Config) *Limiter {
	l := &Limiter{
		rate:     cfg.Rate,
		capacity: cfg.Capacity,
	}
	for i := range l.shards {
		l.shards[i].buckets = make(map[string]*bucket)
	}
	return l
}

// Allow checks if the given key has available tokens.
// Returns true and consumes one token, or false if rate-limited.
func (l *Limiter) Allow(key string) bool {
	sh := l.shard(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	now := time.Now()
	b, ok := sh.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, lastRefill: now}
		sh.buckets[key] = b
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) shard(key string) *limiterShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return &l.shards[h.Sum32()%limiterShards]
}
