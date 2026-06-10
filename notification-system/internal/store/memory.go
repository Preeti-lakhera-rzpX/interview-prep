package store

import (
	"context"
	"hash/fnv"
	"sync"
	"time"

	"interview-prep/internal/model"
)

const numShards = 256

type memoryShard struct {
	mu    sync.RWMutex
	items map[string]*model.Notification
}

// MemoryStore is a concurrent-safe in-memory store using sharded maps.
// Concurrency model: 256 shards, each with its own RWMutex.
// Notification ID is hashed to determine shard.
type MemoryStore struct {
	shards [numShards]memoryShard
}

// NewMemory creates a new sharded in-memory store.
func NewMemory() *MemoryStore {
	s := &MemoryStore{}
	for i := range s.shards {
		s.shards[i].items = make(map[string]*model.Notification)
	}
	return s
}

func (s *MemoryStore) shard(id string) *memoryShard {
	h := fnv.New32a()
	h.Write([]byte(id))
	return &s.shards[h.Sum32()%numShards]
}

// Save persists a new notification. Returns ErrAlreadyExists if ID is taken.
func (s *MemoryStore) Save(_ context.Context, n *model.Notification) error {
	sh := s.shard(n.ID)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if _, exists := sh.items[n.ID]; exists {
		return model.ErrAlreadyExists
	}
	cp := *n
	sh.items[n.ID] = &cp
	return nil
}

// Get retrieves a notification by ID.
func (s *MemoryStore) Get(_ context.Context, id string) (*model.Notification, error) {
	sh := s.shard(id)
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	n, ok := sh.items[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *n
	return &cp, nil
}

// UpdateStatus atomically updates status, attempts, and last error.
func (s *MemoryStore) UpdateStatus(_ context.Context, id string, status model.Status, attempts int, lastErr string) error {
	sh := s.shard(id)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	n, ok := sh.items[id]
	if !ok {
		return model.ErrNotFound
	}
	n.Status = status
	n.Attempts = attempts
	n.LastError = lastErr
	n.UpdatedAt = time.Now()
	return nil
}
