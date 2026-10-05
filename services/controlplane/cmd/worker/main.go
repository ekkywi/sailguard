package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/config"
	"github.com/ekkywi/sailguard/services/controlplane/internal/queue"
	"github.com/ekkywi/sailguard/services/controlplane/internal/redisx"
)

func main() {
	cfg := config.Load()
	log.Printf("sailguard-worker starting (env=%s redis=%s)", cfg.Env, cfg.RedisURL)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	rdb, err := redisx.Connect(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()
	log.Println("redis connected")

	q := queue.New(rdb)

	if err := q.EnsureGroup(ctx, queue.StreamEvents, queue.GroupEvents); err != nil {
		log.Fatalf("ensure group: %v", err)
	}
	log.Printf("consumer group ready: %s / %s", queue.StreamEvents, queue.GroupEvents)

	consumer := "worker-1"

	for {
		if ctx.Err() != nil {
			log.Println("sailguard-worker stopped")
			os.Exit(0)
		}

		msgs, err := q.ReadGroup(ctx, queue.StreamEvents, queue.GroupEvents, consumer, 10)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("sailguard-worker stopped")
				os.Exit(0)
			}
			log.Printf("read group: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if len(msgs) == 0 {
			continue
		}

		for _, m := range msgs {
			log.Printf("got message id=%s values=%v", m.ID, m.Values)
			if err := q.Ack(ctx, queue.StreamEvents, queue.GroupEvents, m.ID); err != nil {
				log.Printf("ack %s: %v", m.ID, err)
			}
		}
	}
}
