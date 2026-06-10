package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"interview-prep/internal/api"
	"interview-prep/internal/dedup"
	"interview-prep/internal/dispatcher"
	"interview-prep/internal/model"
	"interview-prep/internal/provider"
	"interview-prep/internal/queue"
	"interview-prep/internal/ratelimit"
	"interview-prep/internal/retry"
	"interview-prep/internal/service"
	"interview-prep/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Infrastructure
	st := store.NewMemory()
	dd := dedup.New(5 * time.Minute)
	defer dd.Close()

	rl := ratelimit.New(ratelimit.Config{
		Rate:     100,
		Capacity: 200,
	})

	q := queue.New(queue.Config{
		CriticalSize: 10_000,
		HighSize:     50_000,
		NormalSize:   100_000,
		LowSize:      50_000,
	})

	// Providers
	providers := []provider.Provider{
		provider.NewEmail(provider.EmailConfig{
			Host: envOr("SMTP_HOST", "localhost"),
			Port: 587,
			From: envOr("SMTP_FROM", "noreply@example.com"),
		}),
		provider.NewSMS(provider.SMSConfig{
			APIKey: envOr("SMS_API_KEY", ""),
			From:   envOr("SMS_FROM", "+10000000000"),
		}),
		provider.NewPush(provider.PushConfig{
			ServerKey: envOr("PUSH_SERVER_KEY", ""),
			ProjectID: envOr("PUSH_PROJECT_ID", ""),
		}),
		provider.NewWebhook(provider.WebhookConfig{
			Timeout: 10 * time.Second,
		}),
		provider.NewInApp(provider.InAppConfig{
			MaxStoredPerUser: 100,
		}),
	}

	// Retry engine
	re := retry.NewEngine(q, st, retry.Policy{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    60 * time.Second,
		Multiplier:  2.0,
	})
	defer re.Close()

	// Dispatcher
	disp := dispatcher.New(q, providers, re, st, dispatcher.Config{
		Pools: map[model.Channel]dispatcher.PoolConfig{
			model.ChannelEmail:   {MinWorkers: 64, MaxWorkers: 512, QueueSize: 10_000},
			model.ChannelSMS:     {MinWorkers: 32, MaxWorkers: 256, QueueSize: 5_000},
			model.ChannelPush:    {MinWorkers: 128, MaxWorkers: 1024, QueueSize: 20_000},
			model.ChannelWebhook: {MinWorkers: 32, MaxWorkers: 256, QueueSize: 5_000},
			model.ChannelInApp:   {MinWorkers: 64, MaxWorkers: 512, QueueSize: 10_000},
		},
	})
	disp.Start(ctx)

	// Service + API
	svc := service.NewNotifier(st, dd, rl, q)
	handler := api.NewHandler(svc, st, q)

	addr := envOr("LISTEN_ADDR", ":8080")
	srv := api.NewServer(addr, handler)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		log.Println("shutting down...")
	case err := <-errCh:
		return err
	}

	// Graceful shutdown sequence
	srv.Shutdown(15 * time.Second)
	q.Close()
	disp.Drain()
	log.Println("shutdown complete")
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
