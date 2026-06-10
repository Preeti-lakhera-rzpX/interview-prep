package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"kv-cache/internal/api"
	"kv-cache/internal/cache"
	"kv-cache/internal/model"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := loadConfig()

	c, err := cache.New(cfg)
	if err != nil {
		return err
	}

	handler := api.NewHandler(c)
	srv := api.NewServer(cfg.ListenAddr, handler)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("kvcached listening on %s", cfg.ListenAddr)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutting down...")
	case err := <-errCh:
		return err
	}

	if err := srv.Shutdown(15 * time.Second); err != nil {
		log.Printf("server shutdown: %v", err)
	}
	if err := c.Close(); err != nil {
		log.Printf("cache close: %v", err)
	}

	log.Println("shutdown complete")
	return nil
}

func loadConfig() model.Config {
	cfg := model.DefaultConfig()
	cfg.ListenAddr = envOr("LISTEN_ADDR", ":8080")
	cfg.WALPath = envOr("WAL_PATH", "kvcache.wal")

	if v := envOr("WAL_ENABLED", "false"); v == "true" || v == "1" {
		cfg.WALEnabled = true
	}
	if v, err := strconv.Atoi(envOr("MAX_ENTRIES", "100000")); err == nil {
		cfg.MaxEntries = v
	}
	if v, err := strconv.Atoi(envOr("SHARD_COUNT", "64")); err == nil {
		cfg.ShardCount = v
	}
	if v := envOr("EVICTION_POLICY", "lru"); v != "" {
		cfg.EvictionPolicy = model.EvictionPolicy(v)
	}
	if v, err := strconv.Atoi(envOr("DEFAULT_TTL_MS", "0")); err == nil && v > 0 {
		cfg.DefaultTTL = time.Duration(v) * time.Millisecond
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
