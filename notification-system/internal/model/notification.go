package model

import "time"

// Priority controls processing order in the queue.
type Priority int

const (
	PriorityCritical Priority = iota
	PriorityHigh
	PriorityNormal
	PriorityLow
)

// Channel represents a delivery channel.
type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSMS     Channel = "sms"
	ChannelPush    Channel = "push"
	ChannelWebhook Channel = "webhook"
	ChannelInApp   Channel = "inapp"
)

// ValidChannels is the set of recognized channels.
var ValidChannels = map[Channel]bool{
	ChannelEmail:   true,
	ChannelSMS:     true,
	ChannelPush:    true,
	ChannelWebhook: true,
	ChannelInApp:   true,
}

// Status tracks the lifecycle of a notification.
type Status string

const (
	StatusPending    Status = "pending"
	StatusQueued     Status = "queued"
	StatusDispatched Status = "dispatched"
	StatusDelivered  Status = "delivered"
	StatusFailed     Status = "failed"
	StatusRetrying   Status = "retrying"
)

// Payload holds channel-specific content.
type Payload struct {
	Subject string            `json:"subject,omitempty"`
	Body    string            `json:"body"`
	To      string            `json:"to"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Notification is the core domain entity.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Channel   Channel   `json:"channel"`
	Priority  Priority  `json:"priority"`
	Payload   Payload   `json:"payload"`
	Status    Status    `json:"status"`
	Attempts  int       `json:"attempts"`
	MaxRetry  int       `json:"max_retry"`
	LastError string    `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
