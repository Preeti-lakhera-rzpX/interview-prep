package retry

import (
	"context"
	"math"
	"math/rand"
	"sync"
	"time"

	"interview-prep/internal/model"
	"interview-prep/internal/queue"
	"interview-prep/internal/store"
)

// Policy defines retry behavior for failed notifications.
type Policy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

// Engine manages retry scheduling using timers.
// Concurrency model: internal mutex protects the pending map.
// Timers fire and re-enqueue notifications into the priority queue.
type Engine struct {
	queue   *queue.PriorityQueue
	store   store.Store
	policy  Policy
	mu      sync.Mutex
	pending map[string]*time.Timer
	done    chan struct{}
}

// NewEngine creates a retry Engine.
func NewEngine(q *queue.PriorityQueue, st store.Store, p Policy) *Engine {
	return &Engine{
		queue:   q,
		store:   st,
		policy:  p,
		pending: make(map[string]*time.Timer),
		done:    make(chan struct{}),
	}
}

// Schedule registers a failed notification for retry.
// If max attempts exceeded, marks as StatusFailed in the store.
func (e *Engine) Schedule(ctx context.Context, n *model.Notification) error {
	if n.Attempts >= e.policy.MaxAttempts {
		return e.store.UpdateStatus(ctx, n.ID, model.StatusFailed, n.Attempts, n.LastError)
	}

	if err := e.store.UpdateStatus(ctx, n.ID, model.StatusRetrying, n.Attempts, n.LastError); err != nil {
		return err
	}

	delay := e.nextDelay(n.Attempts)

	e.mu.Lock()
	defer e.mu.Unlock()

	select {
	case <-e.done:
		return model.ErrShuttingDown
	default:
	}

	timer := time.AfterFunc(delay, func() {
		e.mu.Lock()
		delete(e.pending, n.ID)
		e.mu.Unlock()

		_ = e.queue.Enqueue(context.Background(), n)
	})
	e.pending[n.ID] = timer
	return nil
}

// Close stops all pending timers and prevents new scheduling.
func (e *Engine) Close() {
	close(e.done)
	e.mu.Lock()
	defer e.mu.Unlock()
	for id, t := range e.pending {
		t.Stop()
		delete(e.pending, id)
	}
}

// PendingCount returns the number of notifications awaiting retry.
func (e *Engine) PendingCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.pending)
}

func (e *Engine) nextDelay(attempt int) time.Duration {
	delay := float64(e.policy.BaseDelay) * math.Pow(e.policy.Multiplier, float64(attempt))
	if delay > float64(e.policy.MaxDelay) {
		delay = float64(e.policy.MaxDelay)
	}
	// Full jitter: uniform random in [0, delay]
	jittered := time.Duration(rand.Float64() * delay)
	if jittered == 0 {
		jittered = e.policy.BaseDelay
	}
	return jittered
}
