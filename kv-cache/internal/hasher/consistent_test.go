package hasher

import (
	"fmt"
	"math"
	"testing"
)

func TestRing_Deterministic(t *testing.T) {
	r := NewRing(16, 128)
	key := "test-key-123"
	shard1 := r.Shard(key)
	shard2 := r.Shard(key)
	if shard1 != shard2 {
		t.Errorf("same key produced different shards: %d vs %d", shard1, shard2)
	}
}

func TestRing_ValidRange(t *testing.T) {
	shardCount := 16
	r := NewRing(shardCount, 128)

	for i := 0; i < 10000; i++ {
		key := fmt.Sprintf("key-%d", i)
		shard := r.Shard(key)
		if shard < 0 || shard >= shardCount {
			t.Fatalf("Shard(%q) = %d, out of range [0, %d)", key, shard, shardCount)
		}
	}
}

func TestRing_Distribution(t *testing.T) {
	shardCount := 16
	r := NewRing(shardCount, 128)
	counts := make([]int, shardCount)
	numKeys := 100000

	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		counts[r.Shard(key)]++
	}

	expected := float64(numKeys) / float64(shardCount)
	// Allow 30% deviation from perfect distribution
	maxDev := expected * 0.30

	for i, count := range counts {
		dev := math.Abs(float64(count) - expected)
		if dev > maxDev {
			t.Errorf("shard %d has %d keys (expected ~%.0f, deviation %.1f%%)",
				i, count, expected, (dev/expected)*100)
		}
	}
}

func TestRing_SingleShard(t *testing.T) {
	r := NewRing(1, 128)
	for i := 0; i < 100; i++ {
		if shard := r.Shard(fmt.Sprintf("key-%d", i)); shard != 0 {
			t.Errorf("single-shard ring returned shard %d", shard)
		}
	}
}
