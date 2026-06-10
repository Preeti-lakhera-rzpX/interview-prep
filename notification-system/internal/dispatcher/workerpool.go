package dispatcher

import (
	"context"
	"log"
	"sync"
	"time"

	"interview-prep/internal/model"
	"interview-prep/internal/provider"
	"interview-prep/internal/retry"
	"interview-prep/internal/store"
)

// PoolConfig controls a per-channel worker pool.
type PoolConfig struct {
	MinWorkers int
	MaxWorkers int
	QueueSize  int
}

// WorkerPool manages a bounded set of workers for a single channel.
// Concurrency model: workers are long-lived goroutines reading from the jobs channel.
// The pool tracks active workers via a WaitGroup for graceful shutdown.
type WorkerPool struct {
	channel  model.Channel
	provider provider.Provider
	retry    *retry.Engine
	store    store.Store
	cfg      PoolConfig
	jobs     chan *model.Notification
	wg       sync.WaitGroup
	done     chan struct{}
}

// NewWorkerPool creates a worker pool for the given channel.
func NewWorkerPool(prov provider.Provider, re *retry.Engine, st store.Store, cfg PoolConfig) *WorkerPool {
	return &WorkerPool{
		channel:  prov.Channel(),
		provider: prov,
		retry:    re,
		store:    st,
		cfg:      cfg,
		jobs:     make(chan *model.Notification, cfg.QueueSize),
		done:     make(chan struct{}),
	}
}

// Start launches the minimum number of workers.
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.cfg.MinWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
	p.wg.Add(1)
	go p.scaler(ctx)
}

// Submit sends a notification to this pool's job queue.
// Returns false if the pool queue is full (backpressure).
func (p *WorkerPool) Submit(n *model.Notification) bool {
	select {
	case p.jobs <- n:
		return true
	default:
		return false
	}
}

// Drain closes the jobs channel and waits for all workers to finish.
func (p *WorkerPool) Drain() {
	close(p.done)
	close(p.jobs)
	p.wg.Wait()
}

func (p *WorkerPool) worker(ctx context.Context) {
	defer p.wg.Done()

	for n := range p.jobs {
		sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := p.provider.Send(sendCtx, n)
		cancel()

		if err != nil {
			n.Attempts++
			n.LastError = err.Error()
			if schedErr := p.retry.Schedule(ctx, n); schedErr != nil {
				log.Printf("[%s] retry schedule failed id=%s: %v", p.channel, n.ID, schedErr)
			}
			_ = p.store.UpdateStatus(ctx, n.ID, model.StatusRetrying, n.Attempts, n.LastError)
		} else {
			_ = p.store.UpdateStatus(ctx, n.ID, model.StatusDelivered, n.Attempts, "")
		}
	}
}

// scaler monitors queue depth and spawns additional workers up to MaxWorkers.
func (p *WorkerPool) scaler(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	currentWorkers := p.cfg.MinWorkers

	for {
		select {
		case <-p.done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			depth := len(p.jobs)
			threshold := p.cfg.QueueSize / 2
			if depth > threshold && currentWorkers < p.cfg.MaxWorkers {
				toAdd := min(p.cfg.MaxWorkers-currentWorkers, 16)
				for i := 0; i < toAdd; i++ {
					p.wg.Add(1)
					go p.worker(ctx)
				}
				currentWorkers += toAdd
				log.Printf("[%s] scaled up to %d workers (depth=%d)", p.channel, currentWorkers, depth)
			}
		}
	}
}
