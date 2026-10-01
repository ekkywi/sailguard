package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("sailguard-worker starting (env=%s redis=%s)", cfg.Env, cfg.RedisURL)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Skeleton: Redis Streams consumers (events/alerts) land in later milestones.
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("sailguard-worker stopped")
			os.Exit(0)
		case <-ticker.C:
			log.Println("worker heartbeat (no consumers registered yet)")
		}
	}
}
