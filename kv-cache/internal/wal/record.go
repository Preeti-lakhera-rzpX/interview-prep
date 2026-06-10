package wal

import "time"

// OpType identifies the type of WAL operation.
type OpType byte

const (
	OpSet    OpType = 1
	OpDelete OpType = 2
)

// Record is a single WAL entry.
type Record struct {
	Op        OpType
	Key       string
	Value     []byte
	ExpiresAt time.Time
}
