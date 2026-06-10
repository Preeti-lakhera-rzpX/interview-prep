package provider

import (
	"context"
	"fmt"
	"log"

	"interview-prep/internal/model"
)

// EmailConfig holds SMTP connection settings.
type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// EmailProvider delivers notifications via email.
type EmailProvider struct {
	cfg EmailConfig
}

// NewEmail creates an EmailProvider with the given config.
func NewEmail(cfg EmailConfig) *EmailProvider {
	return &EmailProvider{cfg: cfg}
}

func (p *EmailProvider) Channel() model.Channel {
	return model.ChannelEmail
}

func (p *EmailProvider) Send(ctx context.Context, n *model.Notification) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if n.Payload.To == "" {
		return fmt.Errorf("email provider: %w: missing recipient", model.ErrInvalidInput)
	}
	log.Printf("[email] sending to=%s subject=%q id=%s", n.Payload.To, n.Payload.Subject, n.ID)
	// Production: use net/smtp or a third-party SMTP client here.
	return nil
}
