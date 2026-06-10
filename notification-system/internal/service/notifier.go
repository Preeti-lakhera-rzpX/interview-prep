package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"interview-prep/internal/dedup"
	"interview-prep/internal/model"
	"interview-prep/internal/queue"
	"interview-prep/internal/ratelimit"
	"interview-prep/internal/store"
)

// Notifier orchestrates the notification pipeline: validate → dedup → rate-limit → enqueue.
type Notifier struct {
	store   store.Store
	dedup   *dedup.Deduplicator
	limiter *ratelimit.Limiter
	queue   *queue.PriorityQueue
}

// NewNotifier creates a Notifier with its dependencies.
func NewNotifier(st store.Store, dd *dedup.Deduplicator, rl *ratelimit.Limiter, q *queue.PriorityQueue) *Notifier {
	return &Notifier{
		store:   st,
		dedup:   dd,
		limiter: rl,
		queue:   q,
	}
}

// SubmitRequest is the input for submitting a notification.
type SubmitRequest struct {
	UserID   string         `json:"user_id"`
	Channel  model.Channel  `json:"channel"`
	Priority model.Priority `json:"priority"`
	Payload  model.Payload  `json:"payload"`
}

// SubmitResponse is returned after successful submission.
type SubmitResponse struct {
	ID     string       `json:"id"`
	Status model.Status `json:"status"`
}

// Submit validates, deduplicates, rate-limits, and enqueues a notification.
func (s *Notifier) Submit(ctx context.Context, req SubmitRequest) (*SubmitResponse, error) {
	if err := s.validate(req); err != nil {
		return nil, err
	}

	if !s.limiter.Allow(fmt.Sprintf("user:%s", req.UserID)) {
		return nil, model.ErrRateLimited
	}
	if !s.limiter.Allow(fmt.Sprintf("channel:%s", req.Channel)) {
		return nil, model.ErrRateLimited
	}

	if s.dedup.IsDuplicate(req.UserID, req.Channel, req.Payload) {
		return nil, model.ErrDuplicate
	}

	now := time.Now()
	n := &model.Notification{
		ID:        generateID(),
		UserID:    req.UserID,
		Channel:   req.Channel,
		Priority:  req.Priority,
		Payload:   req.Payload,
		Status:    model.StatusPending,
		MaxRetry:  5,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Save(ctx, n); err != nil {
		return nil, fmt.Errorf("save notification: %w", err)
	}

	if err := s.queue.Enqueue(ctx, n); err != nil {
		return nil, err
	}

	_ = s.store.UpdateStatus(ctx, n.ID, model.StatusQueued, 0, "")

	return &SubmitResponse{ID: n.ID, Status: model.StatusQueued}, nil
}

func (s *Notifier) validate(req SubmitRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("%w: user_id is required", model.ErrInvalidInput)
	}
	if !model.ValidChannels[req.Channel] {
		return fmt.Errorf("%w: invalid channel %q", model.ErrInvalidInput, req.Channel)
	}
	if req.Payload.To == "" && req.Channel != model.ChannelInApp {
		return fmt.Errorf("%w: payload.to is required for channel %s", model.ErrInvalidInput, req.Channel)
	}
	if req.Payload.Body == "" {
		return fmt.Errorf("%w: payload.body is required", model.ErrInvalidInput)
	}
	return nil
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "notif_" + hex.EncodeToString(b)
}
