package provider

import (
	"context"
	"fmt"
	"log"

	"interview-prep/internal/model"
)

// PushConfig holds push notification service settings.
type PushConfig struct {
	ServerKey string
	ProjectID string
}

// PushProvider delivers push notifications (FCM/APNs).
type PushProvider struct {
	cfg PushConfig
}

// NewPush creates a PushProvider with the given config.
func NewPush(cfg PushConfig) *PushProvider {
	return &PushProvider{cfg: cfg}
}

func (p *PushProvider) Channel() model.Channel {
	return model.ChannelPush
}

func (p *PushProvider) Send(ctx context.Context, n *model.Notification) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if n.Payload.To == "" {
		return fmt.Errorf("push provider: %w: missing device token", model.ErrInvalidInput)
	}
	log.Printf("[push] sending to=%s id=%s", n.Payload.To, n.ID)
	// Production: call FCM/APNs API here.
	return nil
}
