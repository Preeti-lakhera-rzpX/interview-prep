package dispatcher

import (
	"context"
	"log"
	"sync"

	"interview-prep/internal/model"
	"interview-prep/internal/provider"
	"interview-prep/internal/queue"
	"interview-prep/internal/retry"
	"interview-prep/internal/store"
)

// Config maps each channel to its pool configuration.
type Config struct {
	Pools map[model.Channel]PoolConfig
}

// Dispatcher reads from the priority queue and fans out to per-channel worker pools.
// Concurrency model: one goroutine reads the queue; writes to per-channel pool channels.
type Dispatcher struct {
	queue *queue.PriorityQueue
	pools map[model.Channel]*WorkerPool
	wg    sync.WaitGroup
}

// New creates a Dispatcher wiring the queue to per-channel worker pools.
func New(q *queue.PriorityQueue, providers []provider.Provider, re *retry.Engine, st store.Store, cfg Config) *Dispatcher {
	pools := make(map[model.Channel]*WorkerPool, len(providers))
	for _, prov := range providers {
		ch := prov.Channel()
		poolCfg, ok := cfg.Pools[ch]
		if !ok {
			poolCfg = PoolConfig{MinWorkers: 16, MaxWorkers: 128, QueueSize: 5000}
		}
		pools[ch] = NewWorkerPool(prov, re, st, poolCfg)
	}
	return &Dispatcher{
		queue: q,
		pools: pools,
	}
}

// Start launches all worker pools and the dispatch loop.
func (d *Dispatcher) Start(ctx context.Context) {
	for _, pool := range d.pools {
		pool.Start(ctx)
	}
	d.wg.Add(1)
	go d.loop(ctx)
}

// Drain waits for the dispatch loop and all pools to finish.
func (d *Dispatcher) Drain() {
	d.wg.Wait()
	for _, pool := range d.pools {
		pool.Drain()
	}
}

func (d *Dispatcher) loop(ctx context.Context) {
	defer d.wg.Done()

	for {
		n, err := d.queue.Dequeue(ctx)
		if err != nil {
			return
		}

		pool, ok := d.pools[n.Channel]
		if !ok {
			log.Printf("[dispatcher] no pool for channel=%s id=%s", n.Channel, n.ID)
			continue
		}

		if !pool.Submit(n) {
			log.Printf("[dispatcher] pool full channel=%s id=%s, re-enqueueing", n.Channel, n.ID)
			_ = d.queue.Enqueue(ctx, n)
		}
	}
}
