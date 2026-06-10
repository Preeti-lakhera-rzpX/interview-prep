package provider

import (
	"context"

	"interview-prep/internal/model"
)

// Provider sends a notification through a specific channel.
// Implementations must be safe for concurrent use.
type Provider interface {
	Channel() model.Channel
	Send(ctx context.Context, n *model.Notification) error
}
