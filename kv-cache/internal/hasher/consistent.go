package hasher

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"sort"
)

// Ring distributes keys across shards using consistent hashing with virtual nodes.
type Ring struct {
	shardCount int
	sorted     []uint32
	mapping    map[uint32]int
}

// NewRing creates a consistent hash ring with the given shard count and virtual nodes per shard.
func NewRing(shardCount, virtualNodes int) *Ring {
	r := &Ring{
		shardCount: shardCount,
		mapping:    make(map[uint32]int, shardCount*virtualNodes),
	}
	for shard := 0; shard < shardCount; shard++ {
		for vn := 0; vn < virtualNodes; vn++ {
			h := hashKey(fmt.Sprintf("shard-%d-vn-%d", shard, vn))
			r.sorted = append(r.sorted, h)
			r.mapping[h] = shard
		}
	}
	sort.Slice(r.sorted, func(i, j int) bool { return r.sorted[i] < r.sorted[j] })
	return r
}

// Shard returns the shard index for the given key.
func (r *Ring) Shard(key string) int {
	if len(r.sorted) == 0 {
		return 0
	}
	h := hashKey(key)
	idx := sort.Search(len(r.sorted), func(i int) bool {
		return r.sorted[i] >= h
	})
	if idx >= len(r.sorted) {
		idx = 0
	}
	return r.mapping[r.sorted[idx]]
}

func hashKey(key string) uint32 {
	b := make([]byte, len(key)+4)
	binary.LittleEndian.PutUint32(b, crc32.ChecksumIEEE([]byte(key)))
	copy(b[4:], key)
	return crc32.ChecksumIEEE(b)
}
