# KV Cache

A production-grade in-memory key-value cache written in Go with zero external dependencies.

## Features

- **Sharded architecture** — consistent hashing with virtual nodes for even key distribution and reduced lock contention.
- **Eviction policies** — pluggable LRU, LFU, and FIFO via strategy pattern.
- **TTL support** — per-key expiration with lazy deletion and background sweeper.
- **Write-ahead log** — optional file-backed persistence for crash recovery.
- **HTTP API** — RESTful interface for all cache operations plus health and stats.
- **Concurrency-safe** — per-shard RWMutex, atomic stats counters, race-free under `-race`.

## Layout

| Path | Purpose |
|------|---------|
| `cmd/kvcached/` | Server entry point |
| `internal/api/` | HTTP server, handler, middleware |
| `internal/cache/` | Cache interface + sharded implementation |
| `internal/eviction/` | LRU, LFU, FIFO policy implementations |
| `internal/hasher/` | Consistent hash ring |
| `internal/model/` | Domain types, config, errors |
| `internal/stats/` | Atomic statistics collector |
| `internal/wal/` | Write-ahead log for persistence |

## Commands

| Command | Description |
|---------|-------------|
| `make build` | Compile all packages |
| `make test` | Run tests |
| `make race` | Run tests with race detector |
| `make cover` | Generate coverage report |
| `make check` | Full gate: lint + race tests |

## Configuration (Environment Variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `MAX_ENTRIES` | `100000` | Maximum cache entries |
| `SHARD_COUNT` | `64` | Number of shards |
| `EVICTION_POLICY` | `lru` | Eviction policy: `lru`, `lfu`, `fifo` |
| `WAL_ENABLED` | `false` | Enable write-ahead log |
| `WAL_PATH` | `kvcache.wal` | WAL file path |
| `DEFAULT_TTL_MS` | `0` | Default TTL in ms (0 = no expiry) |

## API

```
PUT    /cache/{key}   — Set a key (body: {"value":"base64","ttl_ms":5000})
GET    /cache/{key}   — Get a key
DELETE /cache/{key}   — Delete a key
GET    /health        — Health check
GET    /stats         — Cache statistics
```

## Toolchain

- Go 1.23+
- No external dependencies
