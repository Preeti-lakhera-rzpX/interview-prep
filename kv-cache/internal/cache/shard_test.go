package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"kv-cache/internal/model"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock { return &fakeClock{now: t} }
func (c *fakeClock) Now() time.Time       { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func testCache(t *testing.T, opts ...Option) *ShardedCache {
	t.Helper()
	cfg := model.Config{
		MaxEntries:     1000,
		MaxMemoryBytes: 0,
		EvictionPolicy: model.PolicyLRU,
		ShardCount:     4,
		VirtualNodes:   32,
	}
	c, err := New(cfg, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestCache_SetAndGet(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "hello", []byte("world"), 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := c.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(val) != "world" {
		t.Errorf("Get = %q, want %q", val, "world")
	}
}

func TestCache_GetNotFound(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	_, err := c.Get(ctx, "nonexistent")
	if err != model.ErrNotFound {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestCache_Delete(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	c.Set(ctx, "key", []byte("val"), 0)
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := c.Get(ctx, "key")
	if err != model.ErrNotFound {
		t.Errorf("Get after delete: %v, want ErrNotFound", err)
	}
}

func TestCache_DeleteNotFound(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	err := c.Delete(ctx, "nonexistent")
	if err != model.ErrNotFound {
		t.Errorf("Delete error = %v, want ErrNotFound", err)
	}
}

func TestCache_TTLExpiry(t *testing.T) {
	clk := newFakeClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	c := testCache(t, WithClock(clk))
	ctx := context.Background()

	c.Set(ctx, "ephemeral", []byte("data"), 5*time.Second)

	// Before expiry
	val, err := c.Get(ctx, "ephemeral")
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if string(val) != "data" {
		t.Errorf("value = %q, want %q", val, "data")
	}

	// After expiry
	clk.Advance(6 * time.Second)
	_, err = c.Get(ctx, "ephemeral")
	if err != model.ErrNotFound {
		t.Errorf("Get after expiry: %v, want ErrNotFound", err)
	}
}

func TestCache_Overwrite(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	c.Set(ctx, "key", []byte("v1"), 0)
	c.Set(ctx, "key", []byte("v2"), 0)

	val, _ := c.Get(ctx, "key")
	if string(val) != "v2" {
		t.Errorf("Get = %q, want %q", val, "v2")
	}
}

func TestCache_InvalidKey(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	err := c.Set(ctx, "", []byte("val"), 0)
	if err != model.ErrInvalidKey {
		t.Errorf("Set empty key: %v, want ErrInvalidKey", err)
	}
}

func TestCache_Stats(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()

	c.Set(ctx, "a", []byte("1"), 0)
	c.Get(ctx, "a")     // hit
	c.Get(ctx, "miss")  // miss

	s := c.Stats(ctx)
	if s.Hits != 1 {
		t.Errorf("Hits = %d, want 1", s.Hits)
	}
	if s.Misses != 1 {
		t.Errorf("Misses = %d, want 1", s.Misses)
	}
	if s.Entries != 1 {
		t.Errorf("Entries = %d, want 1", s.Entries)
	}
}

func TestCache_Eviction(t *testing.T) {
	cfg := model.Config{
		MaxEntries:     8, // 2 per shard with 4 shards
		EvictionPolicy: model.PolicyLRU,
		ShardCount:     4,
		VirtualNodes:   32,
	}
	c, _ := New(cfg)
	defer c.Close()
	ctx := context.Background()

	// Fill well beyond capacity to trigger evictions
	for i := 0; i < 100; i++ {
		c.Set(ctx, fmt.Sprintf("key-%d", i), []byte("val"), 0)
	}

	s := c.Stats(ctx)
	if s.Evictions == 0 {
		t.Error("expected evictions to occur")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(3)
		go func(id int) {
			defer wg.Done()
			c.Set(ctx, fmt.Sprintf("key-%d", id), []byte("val"), time.Minute)
		}(i)
		go func(id int) {
			defer wg.Done()
			c.Get(ctx, fmt.Sprintf("key-%d", id))
		}(i)
		go func(id int) {
			defer wg.Done()
			c.Delete(ctx, fmt.Sprintf("key-%d", id%100))
		}(i)
	}
	wg.Wait()
}
