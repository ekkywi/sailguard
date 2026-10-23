package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/config"
	"github.com/ekkywi/sailguard/services/controlplane/internal/db"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/telemetry"
	"github.com/ekkywi/sailguard/services/controlplane/internal/queue"
	"github.com/ekkywi/sailguard/services/controlplane/internal/redisx"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	log.Printf("sailguard-worker starting (env=%s redis=%s)", cfg.Env, cfg.RedisURL)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	log.Println("database connected")

	rdb, err := redisx.Connect(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()
	log.Println("redis connected")

	q := queue.New(rdb)
	events := telemetry.NewStore(pool)

	if err := q.EnsureGroup(ctx, queue.StreamEvents, queue.GroupEvents); err != nil {
		log.Fatalf("ensure group: %v", err)
	}
	log.Printf("consumer group ready: %s / %s", queue.StreamEvents, queue.GroupEvents)

	consumer := "worker-1"
	claimStart := "0-0"

	for {
		if ctx.Err() != nil {
			log.Println("sailguard-worker stopped")
			os.Exit(0)
		}

		claimed, next, err := q.AutoClaim(
			ctx,
			queue.StreamEvents,
			queue.GroupEvents,
			consumer,
			queue.DefaultClaimMinIdle,
			claimStart,
			10,
		)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("sailguard-worker stopped")
				os.Exit(0)
			}
			log.Printf("autoclaim: %v", err)
		} else {
			if next != "" {
				claimStart = next
			}
			if claimStart == "0-0" || len(claimed) == 0 {
				claimStart = "0-0"
			}
			for _, m := range claimed {
				log.Printf("reclaimed id=%s", m.ID)
				processStreamMessage(ctx, q, events, m)
			}
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
			processStreamMessage(ctx, q, events, m)
		}
	}
}

func processStreamMessage(ctx context.Context, q *queue.Client, events *telemetry.Store, m redis.XMessage) {
	ack := func() {
		if err := q.Ack(ctx, queue.StreamEvents, queue.GroupEvents, m.ID); err != nil {
			log.Printf("ack %s: %v", m.ID, err)
		}
	}

	msg, err := queue.DecodeViolationMessage(m.Values)
	if err != nil {
		log.Printf("skip poison id=%s: %v", m.ID, err)
		ack()
		return
	}

	deviceID, err := msg.DeviceUUID()
	if err != nil {
		log.Printf("skip bad device_id id=%s: %v", m.ID, err)
		ack()
		return
	}

	et := strings.TrimSpace(msg.EventType)
	if et != "violation_detected" && et != "violation_blocked" {
		log.Printf("skip bad event_type id=%s: %q", m.ID, msg.EventType)
		ack()
		return
	}
	if strings.TrimSpace(msg.ClientEventID) == "" || strings.TrimSpace(msg.ProcessName) == "" {
		log.Printf("skip incomplete payload id=%s", m.ID)
		ack()
		return
	}

	in := telemetry.InsertViolationInput{
		DeviceID:           deviceID,
		ClientEventID:      msg.ClientEventID,
		EventType:          et,
		OccurredAt:         msg.OccurredAtTime(),
		DedupeKey:          msg.DedupeKey,
		ProcessPID:         msg.ProcessPID,
		ProcessName:        msg.ProcessName,
		ProcessPath:        msg.ProcessPath,
		ProcessHashSHA256:  msg.ProcessHashSHA256,
		ProcessPublisher:   msg.ProcessPublisher,
		ProcessProductName: msg.ProcessProductName,
		ProcessUserName:    msg.ProcessUserName,
		MatchRuleID:        msg.MatchRuleID,
		MatchAction:        msg.MatchAction,
		MatchRuleType:      msg.MatchRuleType,
		MatchValue:         msg.MatchValue,
		MatchCategoryCode:  msg.MatchCategoryCode,
		PolicyVersion:      msg.PolicyVersion,
		PolicyMode:         msg.PolicyMode,
	}

	// Retry in-process for short DB blips; longer failures stay in the PEL
	// until AutoClaim reclaims them (DefaultClaimMinIdle).
	for attempt := 1; attempt <= 3; attempt++ {
		inserted, id, err := events.InsertViolation(ctx, in)
		if err != nil {
			log.Printf("persist id=%s client_event_id=%s attempt=%d: %v", m.ID, msg.ClientEventID, attempt, err)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return
		}
		log.Printf(
			"persisted id=%s client_event_id=%s inserted=%v row=%s",
			m.ID, msg.ClientEventID, inserted, id,
		)
		ack()
		return
	}
}
