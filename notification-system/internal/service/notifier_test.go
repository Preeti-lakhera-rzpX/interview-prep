package service

import (
	"context"
	"testing"
	"time"

	"interview-prep/internal/dedup"
	"interview-prep/internal/model"
	"interview-prep/internal/queue"
	"interview-prep/internal/ratelimit"
	"interview-prep/internal/store"
)

func newTestNotifier() *Notifier {
	st := store.NewMemory()
	dd := dedup.New(1 * time.Minute)
	rl := ratelimit.New(ratelimit.Config{Rate: 100, Capacity: 100})
	q := queue.New(queue.Config{CriticalSize: 100, HighSize: 100, NormalSize: 100, LowSize: 100})
	return NewNotifier(st, dd, rl, q)
}

func TestNotifier_SubmitSuccess(t *testing.T) {
	svc := newTestNotifier()
	ctx := context.Background()

	resp, err := svc.Submit(ctx, SubmitRequest{
		UserID:   "u1",
		Channel:  model.ChannelEmail,
		Priority: model.PriorityNormal,
		Payload:  model.Payload{To: "a@b.com", Body: "hello"},
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if resp.ID == "" {
		t.Error("Submit: expected non-empty ID")
	}
	if resp.Status != model.StatusQueued {
		t.Errorf("Submit: got status %s, want queued", resp.Status)
	}
}

func TestNotifier_SubmitValidation(t *testing.T) {
	svc := newTestNotifier()
	ctx := context.Background()

	tests := []struct {
		name string
		req  SubmitRequest
	}{
		{"missing user_id", SubmitRequest{Channel: model.ChannelEmail, Payload: model.Payload{To: "x", Body: "y"}}},
		{"invalid channel", SubmitRequest{UserID: "u1", Channel: "carrier_pigeon", Payload: model.Payload{To: "x", Body: "y"}}},
		{"missing to", SubmitRequest{UserID: "u1", Channel: model.ChannelSMS, Payload: model.Payload{Body: "y"}}},
		{"missing body", SubmitRequest{UserID: "u1", Channel: model.ChannelEmail, Payload: model.Payload{To: "x"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Submit(ctx, tc.req)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestNotifier_SubmitDuplicate(t *testing.T) {
	svc := newTestNotifier()
	ctx := context.Background()

	req := SubmitRequest{
		UserID:  "u1",
		Channel: model.ChannelEmail,
		Payload: model.Payload{To: "a@b.com", Body: "hello"},
	}

	_, _ = svc.Submit(ctx, req)
	_, err := svc.Submit(ctx, req)
	if err != model.ErrDuplicate {
		t.Errorf("second submit: got %v, want ErrDuplicate", err)
	}
}

func TestNotifier_InAppDoesNotRequireTo(t *testing.T) {
	svc := newTestNotifier()
	ctx := context.Background()

	_, err := svc.Submit(ctx, SubmitRequest{
		UserID:  "u1",
		Channel: model.ChannelInApp,
		Payload: model.Payload{Body: "you have a message"},
	})
	if err != nil {
		t.Fatalf("in-app without To should succeed: %v", err)
	}
}
