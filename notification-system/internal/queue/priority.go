package queue

import (
	"context"
	"sync/atomic"

	"interview-prep/internal/model"
)

// Config controls buffer sizes for each priority band.
type Config struct {
	CriticalSize int
	HighSize     int
	NormalSize   int
	LowSize      int
}

// PriorityQueue provides multi-priority enqueueing and dequeueing.
// Concurrency model: Enqueue and Dequeue are safe for concurrent use.
// Backpressure: Enqueue returns ErrQueueFull if the band's buffer is at capacity.
type PriorityQueue struct {
	critical chan *model.Notification
	high     chan *model.Notification
	normal   chan *model.Notification
	low      chan *model.Notification
	closed   atomic.Bool
	len      atomic.Int64
}

// New creates a PriorityQueue with the given band sizes.
func New(cfg Config) *PriorityQueue {
	return &PriorityQueue{
		critical: make(chan *model.Notification, cfg.CriticalSize),
		high:     make(chan *model.Notification, cfg.HighSize),
		normal:   make(chan *model.Notification, cfg.NormalSize),
		low:      make(chan *model.Notification, cfg.LowSize),
	}
}

// Enqueue adds a notification to the appropriate priority band.
// Returns ErrQueueFull if at capacity, ErrShuttingDown if closed.
func (q *PriorityQueue) Enqueue(_ context.Context, n *model.Notification) error {
	if q.closed.Load() {
		return model.ErrShuttingDown
	}

	ch := q.band(n.Priority)
	select {
	case ch <- n:
		q.len.Add(1)
		return nil
	default:
		return model.ErrQueueFull
	}
}

// Dequeue blocks until a notification is available, returning highest-priority first.
// Returns ctx.Err() if context is cancelled.
func (q *PriorityQueue) Dequeue(ctx context.Context) (*model.Notification, error) {
	for {
		// Non-blocking priority sweep: critical > high > normal > low
		select {
		case n := <-q.critical:
			q.len.Add(-1)
			return n, nil
		default:
		}
		select {
		case n := <-q.high:
			q.len.Add(-1)
			return n, nil
		default:
		}
		select {
		case n := <-q.normal:
			q.len.Add(-1)
			return n, nil
		default:
		}
		select {
		case n := <-q.low:
			q.len.Add(-1)
			return n, nil
		default:
		}

		// Nothing available — block until something arrives or context cancels.
		select {
		case n := <-q.critical:
			q.len.Add(-1)
			return n, nil
		case n := <-q.high:
			q.len.Add(-1)
			return n, nil
		case n := <-q.normal:
			q.len.Add(-1)
			return n, nil
		case n := <-q.low:
			q.len.Add(-1)
			return n, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Len returns the total number of items across all priority bands.
func (q *PriorityQueue) Len() int {
	return int(q.len.Load())
}

// Close prevents new items from being enqueued.
func (q *PriorityQueue) Close() {
	q.closed.Store(true)
}

func (q *PriorityQueue) band(p model.Priority) chan *model.Notification {
	switch p {
	case model.PriorityCritical:
		return q.critical
	case model.PriorityHigh:
		return q.high
	case model.PriorityNormal:
		return q.normal
	default:
		return q.low
	}
}
