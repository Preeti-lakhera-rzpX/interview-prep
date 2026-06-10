package model

import "errors"

var (
	ErrNotFound      = errors.New("notification not found")
	ErrAlreadyExists = errors.New("notification already exists")
	ErrQueueFull     = errors.New("queue is at capacity")
	ErrShuttingDown  = errors.New("system is shutting down")
	ErrRateLimited   = errors.New("rate limit exceeded")
	ErrDuplicate     = errors.New("duplicate notification")
	ErrInvalidInput  = errors.New("invalid input")
)
