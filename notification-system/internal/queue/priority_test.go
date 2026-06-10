package queue

import (
	"context"
	"sync"
	"testing"

	"interview-prep/internal/model"
)

func TestPriorityQueue_EnqueueDequeue(t *testing.T) {
	q := New(Config{CriticalSize: 10, HighSize: 10, NormalSize: 10, LowSize: 10})
	ctx := context.Background()

	notifications := []*model.Notification{
		{ID: "low-1", Priority: model.PriorityLow},
		{ID: "norm-1", Priority: model.PriorityNormal},
		{ID: "high-1", Priority: model.PriorityHigh},
		{ID: "crit-1", Priority: model.PriorityCritical},
	}

	for _, n := range notifications {
		if err := q.Enqueue(ctx, n); err != nil {
			t.Fatalf("Enqueue %s: %v", n.ID, err)
		}
	}

	if q.Len() != 4 {
		t.Fatalf("Len: got %d, want 4", q.Len())
	}

	// Should dequeue in priority order: critical, high, normal, low
	expected := []string{"crit-1", "high-1", "norm-1", "low-1"}
	for _, wantID := range expected {
		got, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue: %v", err)
		}
		if got.ID != wantID {
			t.Errorf("Dequeue: got %s, want %s", got.ID, wantID)
		}
	}
}

func TestPriorityQueue_Backpressure(t *testing.T) {
	q := New(Config{CriticalSize: 1, HighSize: 1, NormalSize: 1, LowSize: 1})
	ctx := context.Background()

	n1 := &model.Notification{ID: "1", Priority: model.PriorityCritical}
	n2 := &model.Notification{ID: "2", Priority: model.PriorityCritical}

	_ = q.Enqueue(ctx, n1)
	err := q.Enqueue(ctx, n2)
	if err != model.ErrQueueFull {
		t.Errorf("Enqueue full: got %v, want ErrQueueFull", err)
	}
}

func TestPriorityQueue_ContextCancellation(t *testing.T) {
	q := New(Config{CriticalSize: 10, HighSize: 10, NormalSize: 10, LowSize: 10})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := q.Dequeue(ctx)
	if err != context.Canceled {
		t.Errorf("Dequeue cancelled: got %v, want context.Canceled", err)
	}
}

func TestPriorityQueue_ShuttingDown(t *testing.T) {
	q := New(Config{CriticalSize: 10, HighSize: 10, NormalSize: 10, LowSize: 10})
	q.Close()

	err := q.Enqueue(context.Background(), &model.Notification{ID: "x", Priority: model.PriorityNormal})
	if err != model.ErrShuttingDown {
		t.Errorf("Enqueue after close: got %v, want ErrShuttingDown", err)
	}
}

func TestPriorityQueue_ConcurrentEnqueueDequeue(t *testing.T) {
	q := New(Config{CriticalSize: 1000, HighSize: 1000, NormalSize: 1000, LowSize: 1000})
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			n := &model.Notification{ID: string(rune(id)), Priority: model.Priority(id % 4)}
			_ = q.Enqueue(ctx, n)
		}(i)
	}
	wg.Wait()

	drained := 0
	for q.Len() > 0 {
		_, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue during drain: %v", err)
		}
		drained++
	}
	if drained != 500 {
		t.Errorf("drained %d, want 500", drained)
	}
}
