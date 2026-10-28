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
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/alerting"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/telemetry"
	"github.com/ekkywi/sailguard/services/controlplane/internal/queue"
	"github.com/ekkywi/sailguard/services/controlplane/internal/redisx"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func alertBackoff(attemptsAfterFail int) time.Duration {
	switch {
	case attemptsAfterFail <= 1:
		return 30 * time.Second
	case attemptsAfterFail == 2:
		return 2 * time.Minute
	case attemptsAfterFail == 3:
		return 10 * time.Minute
	default:
		return 30 * time.Minute
	}
}

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
	alerts := alerting.NewStore(pool)
	go runEventRetention(ctx, cfg.EventsRetentionDays, events)

	if err := q.EnsureGroup(ctx, queue.StreamEvents, queue.GroupEvents); err != nil {
		log.Fatalf("ensure group: %v", err)
	}
	log.Printf("consumer group ready: %s / %s", queue.StreamEvents, queue.GroupEvents)

	if err := q.EnsureGroup(ctx, queue.StreamAlerts, queue.GroupAlerts); err != nil {
		log.Fatalf("ensure group: %v", err)
	}
	log.Printf("consumer group ready: %s / %s", queue.StreamAlerts, queue.GroupAlerts)

	consumer := "worker-1"
	alertConsumer := "worker-1-alerts"
	claimStart := "0-0"
	alertClaimStart := "0-0"

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
				processStreamMessage(ctx, q, events, alerts, m)
			}
		}

		alertClaimed, alertNext, err := q.AutoClaim(
			ctx,
			queue.StreamAlerts,
			queue.GroupAlerts,
			alertConsumer,
			queue.DefaultClaimMinIdle,
			alertClaimStart,
			10,
		)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("sailguard-worker stopped")
				os.Exit(0)
			}
			log.Printf("alert autoclaim: %v", err)
		} else {
			if alertNext != "" {
				alertClaimStart = alertNext
			}
			if alertClaimStart == "0-0" || len(alertClaimed) == 0 {
				alertClaimStart = "0-0"
			}
			for _, m := range alertClaimed {
				log.Printf("alert reclaimed id=%s", m.ID)
				processAlertMessage(ctx, q, alerts, events, m)
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
		for _, m := range msgs {
			processStreamMessage(ctx, q, events, alerts, m)
		}

		alertMsgs, err := q.ReadGroup(ctx, queue.StreamAlerts, queue.GroupAlerts, alertConsumer, 10)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("sailguard-worker stopped")
				os.Exit(0)
			}
			log.Printf("alert read group: %v", err)
		} else {
			for _, m := range alertMsgs {
				processAlertMessage(ctx, q, alerts, events, m)
			}
		}

		requeueDueAlerts(ctx, q, alerts)

		if len(msgs) == 0 && len(alertMsgs) == 0 {
			continue
		}
	}
}

func processStreamMessage(
	ctx context.Context,
	q *queue.Client,
	events *telemetry.Store,
	alerts *alerting.Store,
	m redis.XMessage,
) {
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
		if inserted {
			enqueueAlerts(ctx, q, alerts, id)
		}
		ack()
		return
	}
}

func enqueueAlerts(
	ctx context.Context,
	q *queue.Client,
	alerts *alerting.Store,
	eventID uuid.UUID,
) {
	channels, err := alerts.ListEnabledWebhooks(ctx)
	if err != nil {
		log.Printf("alert list channels: %v", err)
		return
	}
	if len(channels) == 0 {
		return
	}
	for _, ch := range channels {
		d, err := alerts.EnqueueDelivery(ctx, ch.ID, eventID)
		if err != nil {
			log.Printf("alert enqueue db channel=%s event=%s: %v", ch.ID, eventID, err)
			continue
		}
		if err := produceAlertJob(ctx, q, d); err != nil {
			log.Printf("alert produce delivery=%s: %v", d.ID, err)
			continue
		}
		log.Printf(
			"alert enqueued delivery=%s channel=%s event=%s",
			d.ID, ch.ID, eventID,
		)
	}
}

func produceAlertJob(ctx context.Context, q *queue.Client, d *alerting.Delivery) error {
	values, err := queue.EncodeAlertValues(queue.AlertMessage{
		DeliveryID: d.ID.String(),
		ChannelID:  d.ChannelID.String(),
		EventID:    d.EventID.String(),
	})
	if err != nil {
		return err
	}
	_, err = q.Produce(ctx, queue.StreamAlerts, values)
	return err
}

func requeueDueAlerts(ctx context.Context, q *queue.Client, alerts *alerting.Store) {
	due, err := alerts.ClaimDueRetries(ctx, 20)
	if err != nil {
		log.Printf("alert claim due: %v", err)
		return
	}
	for i := range due {
		d := &due[i]
		if err := produceAlertJob(ctx, q, d); err != nil {
			log.Printf("alert requeue delivery=%s: %v", d.ID, err)
			continue
		}
		log.Printf("alert requeued delivery=%s attempts=%d", d.ID, d.Attempts)
	}
}

func processAlertMessage(
	ctx context.Context,
	q *queue.Client,
	alerts *alerting.Store,
	events *telemetry.Store,
	m redis.XMessage,
) {
	ack := func() {
		if err := q.Ack(ctx, queue.StreamAlerts, queue.GroupAlerts, m.ID); err != nil {
			log.Printf("alert ack %s: %v", m.ID, err)
		}
	}

	msg, err := queue.DecodeAlertMessage(m.Values)
	if err != nil {
		log.Printf("alert skip poison id=%s: %v", m.ID, err)
		ack()
		return
	}
	deliveryID, err := msg.DeliveryUUID()
	if err != nil {
		log.Printf("alert skip bad delivery_id id=%s: %v", m.ID, err)
		ack()
		return
	}

	d, err := alerts.GetDelivery(ctx, deliveryID)
	if err != nil {
		log.Printf("alert get delivery=%s: %v", deliveryID, err)
		ack()
		return
	}
	if d.Status == alerting.DeliveryDelivered || d.Status == alerting.DeliveryDead {
		ack()
		return
	}
	if d.NextRetryAt != nil && d.NextRetryAt.After(time.Now()) {
		ack()
		return
	}

	ch, err := alerts.GetChannel(ctx, d.ChannelID)
	if err != nil {
		log.Printf("alert get channel delivery=%s: %v", deliveryID, err)
		_, _ = alerts.MarkAttemptFailed(ctx, deliveryID, err.Error(), alertBackoff(d.Attempts+1))
		ack()
		return
	}
	url := strings.TrimSpace(ch.WebhookURL())
	if url == "" || !ch.Enabled {
		_, _ = alerts.MarkAttemptFailed(ctx, deliveryID, "channel disabled or empty url", alertBackoff(d.Attempts+1))
		ack()
		return
	}

	ev, err := events.GetViolationByID(ctx, d.EventID)
	if err != nil {
		log.Printf("alert load event delivery=%s: %v", deliveryID, err)
		_, _ = alerts.MarkAttemptFailed(ctx, deliveryID, err.Error(), alertBackoff(d.Attempts+1))
		ack()
		return
	}

	payload := map[string]any{
		"type":            "sailguard.violation",
		"event_id":        ev.ID.String(),
		"device_id":       ev.DeviceID.String(),
		"client_event_id": ev.ClientEventID,
		"event_type":      ev.EventType,
		"occurred_at":     ev.OccurredAt.UTC().Format(time.RFC3339),
		"process_name":    ev.ProcessName,
		"process_path":    ev.ProcessPath,
		"process_user":    ev.ProcessUserName,
		"match_action":    ev.MatchAction,
		"match_value":     ev.MatchValue,
		"policy_mode":     ev.PolicyMode,
	}

	if err := alerting.DeliverWebhook(ctx, url, payload); err != nil {
		log.Printf("alert webhook fail delivery=%s: %v", deliveryID, err)
		if _, markErr := alerts.MarkAttemptFailed(ctx, deliveryID, err.Error(), alertBackoff(d.Attempts+1)); markErr != nil {
			log.Printf("alert mark failed delivery=%s: %v", deliveryID, markErr)
		}
		ack()
		return
	}

	if _, err := alerts.MarkDelivered(ctx, deliveryID); err != nil {
		log.Printf("alert mark delivered delivery=%s: %v", deliveryID, err)
		ack()
		return
	}
	log.Printf("alert webhook ok delivery=%s", deliveryID)
	ack()
}

func runEventRetention(ctx context.Context, days int, events *telemetry.Store) {
	if days <= 0 {
		log.Println("event retention disabled (SG_EVENTS_RETENTION_DAYS=0)")
		return
	}
	log.Printf("event retention: keep %d days", days)

	runOnce := func() {
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		var total int64
		for {
			n, err := events.DeleteOlderThan(ctx, cutoff, 1000)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("event retention: %v", err)
				return
			}
			total += n
			if n == 0 {
				break
			}
		}
		if total > 0 {
			log.Printf("event retention: purged %d rows (before %s)", total, cutoff.Format(time.RFC3339))
		}
	}

	runOnce()

	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			runOnce()
		}
	}
}
