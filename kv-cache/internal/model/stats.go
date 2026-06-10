package model

// Stats holds cache runtime statistics.
type Stats struct {
	Hits          uint64  `json:"hits"`
	Misses        uint64  `json:"misses"`
	Evictions     uint64  `json:"evictions"`
	Entries       int64   `json:"entries"`
	MemoryBytes   int64   `json:"memory_bytes"`
	HitRatio      float64 `json:"hit_ratio"`
	UptimeSeconds float64 `json:"uptime_seconds"`
}
