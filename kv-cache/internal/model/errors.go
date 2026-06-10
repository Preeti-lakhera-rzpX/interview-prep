package model

import "errors"

var (
	ErrNotFound    = errors.New("key not found")
	ErrKeyExpired  = errors.New("key expired")
	ErrCacheFull   = errors.New("cache at capacity")
	ErrInvalidKey  = errors.New("invalid key")
	ErrInvalidTTL  = errors.New("invalid TTL")
	ErrWALCorrupt  = errors.New("WAL corrupt")
	ErrShuttingDown = errors.New("cache shutting down")
)
