package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	StreamEvents = "sg:events"
	GroupEvents = "sg-workers"
)

type Client struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Client {
	return &Client{rdb: rdb}
}

func (c *Client) Produce(ctx context.Context, stream string, values map[string]any) (string, error) {
	id, err := c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd %s: %w", stream, err)
	}
	return id, nil
}

func (c *Client) EnsureGroup(ctx context.Context, stream, group string) error {
	err := c.rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("xgroup create %s / %s: stream, group, err")
	}
	return nil
}

func (c *Client) ReadGroup(ctx context.Context, stream, group, consumer string, count int64) ([]redis.XMessage, error) {
	streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: group,
		Consumer: consumer,
		Streams: []string{stream, ">"},
		Count: count,
		Block: 2 * time.Second,
	}).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("xreadgroup %s: %w", stream, err)
	}
	if len(streams) == 0 {
		return nil, nil
	}
	return streams[0].Messages, nil
}

func (c *Client) Ack(ctx context.Context, stream, group string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return c.rdb.XAck(ctx, stream, group, ids...).Err()
}