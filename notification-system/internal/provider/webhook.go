package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"interview-prep/internal/model"
)

// WebhookConfig holds HTTP client settings for webhook delivery.
type WebhookConfig struct {
	Timeout time.Duration
}

// WebhookProvider delivers notifications via HTTP callbacks.
type WebhookProvider struct {
	client *http.Client
}

// NewWebhook creates a WebhookProvider with the given config.
func NewWebhook(cfg WebhookConfig) *WebhookProvider {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &WebhookProvider{
		client: &http.Client{Timeout: timeout},
	}
}

func (p *WebhookProvider) Channel() model.Channel {
	return model.ChannelWebhook
}

func (p *WebhookProvider) Send(ctx context.Context, n *model.Notification) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if n.Payload.To == "" {
		return fmt.Errorf("webhook provider: %w: missing URL", model.ErrInvalidInput)
	}

	body, err := json.Marshal(map[string]any{
		"notification_id": n.ID,
		"user_id":         n.UserID,
		"payload":         n.Payload,
	})
	if err != nil {
		return fmt.Errorf("webhook provider: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.Payload.To, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook provider: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[webhook] posting to=%s id=%s", n.Payload.To, n.ID)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook provider: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook provider: received status %d", resp.StatusCode)
	}
	return nil
}
