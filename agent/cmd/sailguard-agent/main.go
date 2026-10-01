package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ekkywi/sailguard/agent/internal/app"
	"github.com/ekkywi/sailguard/agent/internal/config"
	"github.com/ekkywi/sailguard/agent/internal/platform"
)

func main() {
	cfg := config.Load()
	h := platform.NewHost()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("sailguard-agent starting os_family=%s version=%s", h.OSFamily(), cfg.AgentVersion)
	if err := app.Run(ctx, cfg, h); err != nil {
		log.Printf("agent stopped with error: %v", err)
		os.Exit(1)
	}
}
