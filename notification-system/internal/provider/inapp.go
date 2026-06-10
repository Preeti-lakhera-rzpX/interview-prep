package provider

import (
	"context"
	"fmt"
	"log"

	"interview-prep/internal/model"
)

// InAppConfig holds in-app notification settings.
type InAppConfig struct {
	MaxStoredPerUser int
}

// InAppProvider delivers in-app notifications (stored for user retrieval).
type InAppProvider struct {
	cfg InAppConfig
}

// NewInApp creates an InAppProvider with the given config.
func NewInApp(cfg InAppConfig) *InAppProvider {
	return &InAppProvider{cfg: cfg}
}

func (p *InAppProvider) Channel() model.Channel {
	return model.ChannelInApp
}

func (p *InAppProvider) Send(ctx context.Context, n *model.Notification) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if n.UserID == "" {
		return fmt.Errorf("inapp provider: %w: missing user ID", model.ErrInvalidInput)
	}
	log.Printf("[inapp] storing for user=%s id=%s", n.UserID, n.ID)
	// Production: write to a user-facing inbox store (DB, cache, SSE push).
	return nil
}
