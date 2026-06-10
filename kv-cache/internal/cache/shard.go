package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kv-cache/internal/eviction"
	"kv-cache/internal/hasher"
	"kv-cache/internal/model"
	"kv-cache/internal/stats"
	"kv-cache/internal/wal"
)

const entryOverhead = 128 // estimated per-entry memory overhead in bytes

type internalEntry struct {
	value     []byte
	expiresAt time.Time
	createdAt time.Time
}

func (e *internalEntry) memSize(keyLen int) int64 {
	return int64(keyLen + len(e.value) + entryOverhead)
}

// cacheShard is an independently-locked partition of the cache.
// Concurrency: the mu RWMutex guards items, policy, and memUsed.
type cacheShard struct {
	mu      sync.RWMutex
	items   map[string]*internalEntry
	policy  eviction.Policy
	memUsed int64
	maxMem  int64
	clock   Clock
	stats   *stats.Collector
}

// ShardedCache distributes keys across shards via consistent hashing.
type ShardedCache struct {
	shards    []*cacheShard
	ring      *hasher.Ring
	stats     *stats.Collector
	wal       wal.WAL
	clock     Clock
	done      chan struct{}
	closeOnce sync.Once
}

// New creates a ShardedCache from the given config.
// If cfg.WALEnabled, it replays existing WAL records to restore state.
func New(cfg model.Config, opts ...Option) (*ShardedCache, error) {
	o := options{clock: realClock{}}
	for _, opt := range opts {
		opt(&o)
	}

	collector := stats.NewCollector()
	ring := hasher.NewRing(cfg.ShardCount, cfg.VirtualNodes)

	perShardMax := int64(0)
	if cfg.MaxMemoryBytes > 0 {
		perShardMax = cfg.MaxMemoryBytes / int64(cfg.ShardCount)
	}
	perShardEntries := cfg.MaxEntries / cfg.ShardCount
	if perShardEntries < 1 {
		perShardEntries = 1
	}

	shards := make([]*cacheShard, cfg.ShardCount)
	for i := range shards {
		shards[i] = &cacheShard{
			items:  make(map[string]*internalEntry),
			policy: eviction.New(cfg.EvictionPolicy, perShardEntries),
			maxMem: perShardMax,
			clock:  o.clock,
			stats:  collector,
		}
	}

	sc := &ShardedCache{
		shards: shards,
		ring:   ring,
		stats:  collector,
		clock:  o.clock,
		done:   make(chan struct{}),
	}

	if cfg.WALEnabled {
		w, err := wal.Open(cfg.WALPath)
		if err != nil {
			return nil, fmt.Errorf("cache: open wal: %w", err)
		}
		sc.wal = w
		if err := sc.replayWAL(); err != nil {
			w.Close()
			return nil, fmt.Errorf("cache: replay wal: %w", err)
		}
	}

	go sc.sweepExpired()
	return sc, nil
}

func (sc *ShardedCache) Get(ctx context.Context, key string) ([]byte, error) {
	shard := sc.getShard(key)
	shard.mu.RLock()
	entry, ok := shard.items[key]
	shard.mu.RUnlock()

	if !ok {
		sc.stats.RecordMiss()
		return nil, model.ErrNotFound
	}

	if !entry.expiresAt.IsZero() && sc.clock.Now().After(entry.expiresAt) {
		sc.stats.RecordMiss()
		// Lazy expiration
		shard.mu.Lock()
		shard.deleteEntry(key)
		shard.mu.Unlock()
		return nil, model.ErrNotFound
	}

	shard.mu.Lock()
	shard.policy.Access(key)
	shard.mu.Unlock()

	sc.stats.RecordHit()
	val := make([]byte, len(entry.value))
	copy(val, entry.value)
	return val, nil
}

func (sc *ShardedCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if key == "" {
		return model.ErrInvalidKey
	}

	now := sc.clock.Now()
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = now.Add(ttl)
	}

	if sc.wal != nil {
		rec := wal.Record{Op: wal.OpSet, Key: key, Value: value, ExpiresAt: expiresAt}
		if err := sc.wal.Append(ctx, rec); err != nil {
			return fmt.Errorf("cache set wal: %w", err)
		}
	}

	shard := sc.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Remove old entry if exists
	if old, ok := shard.items[key]; ok {
		shard.memUsed -= old.memSize(len(key))
		shard.policy.Remove(key)
		sc.stats.DecrEntries()
	}

	entry := &internalEntry{
		value:     make([]byte, len(value)),
		expiresAt: expiresAt,
		createdAt: now,
	}
	copy(entry.value, value)

	// Evict if needed
	if evicted := shard.policy.Add(key); evicted != "" {
		shard.deleteEntry(evicted)
		sc.stats.RecordEviction()
	}

	shard.items[key] = entry
	shard.memUsed += entry.memSize(len(key))
	sc.stats.IncrEntries()
	sc.stats.AddMemory(entry.memSize(len(key)))
	return nil
}

func (sc *ShardedCache) Delete(ctx context.Context, key string) error {
	if sc.wal != nil {
		rec := wal.Record{Op: wal.OpDelete, Key: key}
		if err := sc.wal.Append(ctx, rec); err != nil {
			return fmt.Errorf("cache delete wal: %w", err)
		}
	}

	shard := sc.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if _, ok := shard.items[key]; !ok {
		return model.ErrNotFound
	}
	shard.deleteEntry(key)
	return nil
}

func (sc *ShardedCache) Stats(_ context.Context) model.Stats {
	return sc.stats.Snapshot()
}

func (sc *ShardedCache) Close() error {
	var err error
	sc.closeOnce.Do(func() {
		close(sc.done)
		if sc.wal != nil {
			err = sc.wal.Close()
		}
	})
	return err
}

func (sc *ShardedCache) getShard(key string) *cacheShard {
	return sc.shards[sc.ring.Shard(key)]
}

// deleteEntry removes a key from shard (caller must hold write lock).
func (s *cacheShard) deleteEntry(key string) {
	entry, ok := s.items[key]
	if !ok {
		return
	}
	mem := entry.memSize(len(key))
	s.memUsed -= mem
	s.stats.SubMemory(mem)
	s.stats.DecrEntries()
	s.policy.Remove(key)
	delete(s.items, key)
}

func (sc *ShardedCache) sweepExpired() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	idx := 0
	for {
		select {
		case <-sc.done:
			return
		case <-ticker.C:
			sc.sweepShard(idx % len(sc.shards))
			idx++
		}
	}
}

func (sc *ShardedCache) sweepShard(idx int) {
	shard := sc.shards[idx]
	now := sc.clock.Now()

	shard.mu.Lock()
	defer shard.mu.Unlock()

	for key, entry := range shard.items {
		if !entry.expiresAt.IsZero() && now.After(entry.expiresAt) {
			shard.deleteEntry(key)
		}
	}
}

func (sc *ShardedCache) replayWAL() error {
	ctx := context.Background()
	now := sc.clock.Now()

	return sc.wal.Replay(ctx, func(r wal.Record) error {
		switch r.Op {
		case wal.OpSet:
			if !r.ExpiresAt.IsZero() && now.After(r.ExpiresAt) {
				return nil // skip expired
			}
			shard := sc.getShard(r.Key)
			shard.mu.Lock()
			entry := &internalEntry{
				value:     r.Value,
				expiresAt: r.ExpiresAt,
				createdAt: now,
			}
			if evicted := shard.policy.Add(r.Key); evicted != "" {
				shard.deleteEntry(evicted)
			}
			shard.items[r.Key] = entry
			shard.memUsed += entry.memSize(len(r.Key))
			sc.stats.IncrEntries()
			sc.stats.AddMemory(entry.memSize(len(r.Key)))
			shard.mu.Unlock()
		case wal.OpDelete:
			shard := sc.getShard(r.Key)
			shard.mu.Lock()
			shard.deleteEntry(r.Key)
			shard.mu.Unlock()
		}
		return nil
	})
}

// Option configures the ShardedCache.
type Option func(*options)

type options struct {
	clock Clock
}

// WithClock sets a custom clock (for testing).
func WithClock(c Clock) Option {
	return func(o *options) { o.clock = c }
}
