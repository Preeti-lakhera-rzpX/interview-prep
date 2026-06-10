package provider

import (
	"context"
	"fmt"
	"log"

	"interview-prep/internal/model"
)

// SMSConfig holds SMS gateway settings.
type SMSConfig struct {
	APIKey    string
	APISecret string
	From      string
}

// SMSProvider delivers notifications via SMS.
type SMSProvider struct {
	cfg SMSConfig
}

// NewSMS creates an SMSProvider with the given config.
func NewSMS(cfg SMSConfig) *SMSProvider {
	return &SMSProvider{cfg: cfg}
}

func (p *SMSProvider) Channel() model.Channel {
	return model.ChannelSMS
}

func (p *SMSProvider) Send(ctx context.Context, n *model.Notification) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if n.Payload.To == "" {
		return fmt.Errorf("sms provider: %w: missing phone number", model.ErrInvalidInput)
	}
	log.Printf("[sms] sending to=%s id=%s", n.Payload.To, n.ID)
	// Production: call Twilio/SNS/other SMS API here.
	return nil
}
