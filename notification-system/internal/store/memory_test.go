package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"interview-prep/internal/model"
)

func TestMemoryStore_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	n := &model.Notification{
		ID:        "test-1",
		UserID:    "u1",
		Channel:   model.ChannelEmail,
		Status:    model.StatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.Save(ctx, n); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}

	got, err := s.Get(ctx, "test-1")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.ID != "test-1" || got.UserID != "u1" {
		t.Errorf("Get: got %+v, want id=test-1 user=u1", got)
	}
}

func TestMemoryStore_SaveDuplicate(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	n := &model.Notification{ID: "dup-1", UserID: "u1", Channel: model.ChannelSMS}
	_ = s.Save(ctx, n)

	err := s.Save(ctx, n)
	if err != model.ErrAlreadyExists {
		t.Errorf("Save duplicate: got %v, want ErrAlreadyExists", err)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	_, err := s.Get(ctx, "nonexistent")
	if err != model.ErrNotFound {
		t.Errorf("Get nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	n := &model.Notification{ID: "upd-1", Status: model.StatusPending}
	_ = s.Save(ctx, n)

	err := s.UpdateStatus(ctx, "upd-1", model.StatusDelivered, 1, "")
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got, _ := s.Get(ctx, "upd-1")
	if got.Status != model.StatusDelivered || got.Attempts != 1 {
		t.Errorf("UpdateStatus: got status=%s attempts=%d", got.Status, got.Attempts)
	}
}

func TestMemoryStore_UpdateStatusNotFound(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	err := s.UpdateStatus(ctx, "ghost", model.StatusFailed, 0, "oops")
	if err != model.ErrNotFound {
		t.Errorf("UpdateStatus nonexistent: got %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := NewMemory()

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			n := &model.Notification{ID: id, UserID: "u", Channel: model.ChannelPush}
			_ = s.Save(ctx, n)
			_, _ = s.Get(ctx, id)
			_ = s.UpdateStatus(ctx, id, model.StatusDelivered, 1, "")
		}(fmt.Sprintf("conc-%d", i))
	}
	wg.Wait()
}
