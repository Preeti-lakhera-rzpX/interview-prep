package stats

import (
	"sync"
	"testing"
)

func TestCollector_BasicCounting(t *testing.T) {
	c := NewCollector()

	c.RecordHit()
	c.RecordHit()
	c.RecordMiss()
	c.RecordEviction()
	c.IncrEntries()
	c.IncrEntries()
	c.DecrEntries()
	c.AddMemory(1024)
	c.SubMemory(512)

	s := c.Snapshot()
	if s.Hits != 2 {
		t.Errorf("Hits = %d, want 2", s.Hits)
	}
	if s.Misses != 1 {
		t.Errorf("Misses = %d, want 1", s.Misses)
	}
	if s.Evictions != 1 {
		t.Errorf("Evictions = %d, want 1", s.Evictions)
	}
	if s.Entries != 1 {
		t.Errorf("Entries = %d, want 1", s.Entries)
	}
	if s.MemoryBytes != 512 {
		t.Errorf("MemoryBytes = %d, want 512", s.MemoryBytes)
	}
}

func TestCollector_HitRatio(t *testing.T) {
	tests := []struct {
		name   string
		hits   int
		misses int
		want   float64
	}{
		{"all hits", 10, 0, 1.0},
		{"all misses", 0, 10, 0.0},
		{"even split", 5, 5, 0.5},
		{"no accesses", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCollector()
			for i := 0; i < tt.hits; i++ {
				c.RecordHit()
			}
			for i := 0; i < tt.misses; i++ {
				c.RecordMiss()
			}
			s := c.Snapshot()
			if s.HitRatio != tt.want {
				t.Errorf("HitRatio = %f, want %f", s.HitRatio, tt.want)
			}
		})
	}
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	c := NewCollector()
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(4)
		go func() { defer wg.Done(); c.RecordHit() }()
		go func() { defer wg.Done(); c.RecordMiss() }()
		go func() { defer wg.Done(); c.IncrEntries() }()
		go func() { defer wg.Done(); c.AddMemory(100) }()
	}
	wg.Wait()

	s := c.Snapshot()
	if s.Hits != 1000 {
		t.Errorf("Hits = %d, want 1000", s.Hits)
	}
	if s.Misses != 1000 {
		t.Errorf("Misses = %d, want 1000", s.Misses)
	}
	if s.Entries != 1000 {
		t.Errorf("Entries = %d, want 1000", s.Entries)
	}
	if s.MemoryBytes != 100000 {
		t.Errorf("MemoryBytes = %d, want 100000", s.MemoryBytes)
	}
}
