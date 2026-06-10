package stats

import (
	"sync/atomic"
	"time"

	"kv-cache/internal/model"
)

// Collector tracks cache statistics using atomic counters.
// All methods are safe for concurrent use.
type Collector struct {
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
	entries   atomic.Int64
	memBytes  atomic.Int64
	startTime time.Time
}

// NewCollector creates a new statistics collector.
func NewCollector() *Collector {
	return &Collector{startTime: time.Now()}
}

func (c *Collector) RecordHit()              { c.hits.Add(1) }
func (c *Collector) RecordMiss()             { c.misses.Add(1) }
func (c *Collector) RecordEviction()         { c.evictions.Add(1) }
func (c *Collector) IncrEntries()            { c.entries.Add(1) }
func (c *Collector) DecrEntries()            { c.entries.Add(-1) }
func (c *Collector) AddMemory(bytes int64)   { c.memBytes.Add(bytes) }
func (c *Collector) SubMemory(bytes int64)   { c.memBytes.Add(-bytes) }

// Snapshot returns a point-in-time copy of all stats.
func (c *Collector) Snapshot() model.Stats {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses

	var ratio float64
	if total > 0 {
		ratio = float64(hits) / float64(total)
	}

	return model.Stats{
		Hits:          hits,
		Misses:        misses,
		Evictions:     c.evictions.Load(),
		Entries:       c.entries.Load(),
		MemoryBytes:   c.memBytes.Load(),
		HitRatio:      ratio,
		UptimeSeconds: time.Since(c.startTime).Seconds(),
	}
}
