package main

import (
	"context"
	"log"
	"os"

	"github.com/ekkywi/sailguard/agent/internal/app"
	"github.com/ekkywi/sailguard/agent/internal/config"
	"github.com/ekkywi/sailguard/agent/internal/platform"
)

func main() {
	err := platform.RunAgent(func(ctx context.Context) error {
		cfg := config.Load()
		h := platform.NewHost()
		log.Printf("sailguard-agent starting os_family=%s version=%s", h.OSFamily(), cfg.AgentVersion)
		return app.Run(ctx, cfg, h)
	})
	if err != nil {
		log.Printf("agent stopped with error: %v", err)
		os.Exit(1)
	}
}
