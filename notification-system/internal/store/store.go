package store

import (
	"context"

	"interview-prep/internal/model"
)

// Store persists and retrieves notification state.
// All methods must be safe for concurrent use.
type Store interface {
	Save(ctx context.Context, n *model.Notification) error
	Get(ctx context.Context, id string) (*model.Notification, error)
	UpdateStatus(ctx context.Context, id string, status model.Status, attempts int, lastErr string) error
}
