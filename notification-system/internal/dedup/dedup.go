package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"hash/fnv"
	"sync"
	"time"

	"interview-prep/internal/model"
)

const dedupShards = 64

type entry struct {
	expiry time.Time
}

type dedupShard struct {
	mu    sync.RWMutex
	items map[string]entry
}

// Deduplicator prevents duplicate notifications within a TTL window.
// Concurrency model: 64 shards with per-shard RWMutex.
// Background goroutine sweeps expired entries every TTL/2.
type Deduplicator struct {
	ttl    time.Duration
	shards [dedupShards]dedupShard
	done   chan struct{}
}

// New creates a Deduplicator with the given TTL and starts the sweeper.
func New(ttl time.Duration) *Deduplicator {
	d := &Deduplicator{
		ttl:  ttl,
		done: make(chan struct{}),
	}
	for i := range d.shards {
		d.shards[i].items = make(map[string]entry)
	}
	go d.sweep()
	return d
}

// IsDuplicate returns true if this notification was seen within the TTL window.
// If not a duplicate, records it and returns false.
func (d *Deduplicator) IsDuplicate(userID string, channel model.Channel, payload model.Payload) bool {
	key := d.key(userID, channel, payload)
	sh := d.shard(key)

	now := time.Now()

	sh.mu.RLock()
	if e, ok := sh.items[key]; ok && now.Before(e.expiry) {
		sh.mu.RUnlock()
		return true
	}
	sh.mu.RUnlock()

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if e, ok := sh.items[key]; ok && now.Before(e.expiry) {
		return true
	}
	sh.items[key] = entry{expiry: now.Add(d.ttl)}
	return false
}

// Close stops the background sweeper.
func (d *Deduplicator) Close() {
	close(d.done)
}

func (d *Deduplicator) sweep() {
	ticker := time.NewTicker(d.ttl / 2)
	defer ticker.Stop()

	for {
		select {
		case <-d.done:
			return
		case now := <-ticker.C:
			for i := range d.shards {
				sh := &d.shards[i]
				sh.mu.Lock()
				for k, e := range sh.items {
					if now.After(e.expiry) {
						delete(sh.items, k)
					}
				}
				sh.mu.Unlock()
			}
		}
	}
}

func (d *Deduplicator) key(userID string, channel model.Channel, payload model.Payload) string {
	h := sha256.New()
	h.Write([]byte(userID))
	h.Write([]byte(channel))
	h.Write([]byte(payload.To))
	h.Write([]byte(payload.Body))
	h.Write([]byte(payload.Subject))
	return hex.EncodeToString(h.Sum(nil))
}

func (d *Deduplicator) shard(key string) *dedupShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return &d.shards[h.Sum32()%dedupShards]
}
