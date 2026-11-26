package app

import (
	"context"
	"log"
	"time"

	"github.com/ekkywi/sailguard/agent/internal/config"
	"github.com/ekkywi/sailguard/agent/internal/netenforce"
	"github.com/ekkywi/sailguard/agent/internal/networkpolicycache"
)

const networkRefreshEvery = 30 * time.Second

func networkRefreshLoop(ctx context.Context, cfg config.Config, dataDir, serverURL string) {
	if !cfg.NetworkEnforce {
		return
	}
	ticker := time.NewTicker(networkRefreshEvery)
	defer ticker.Stop()

	var lastSig string
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			doc, err := networkpolicycache.Load(dataDir)
			if err != nil {
				log.Printf("network refresh: cache load failed: %v", err)
				continue
			}
			if doc == nil {
				continue
			}
			spec := netenforce.SpecFromDocument(*doc)
			sig, err := netenforce.RefreshHosts(ctx, spec, serverURL, lastSig)
			if err != nil {
				log.Printf("network refresh failed: %v", err)
			}
			lastSig = sig
		}
	}
}