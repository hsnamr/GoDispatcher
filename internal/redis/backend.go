package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/halamri/go-dispatcher/internal/config"
)

const (
	BufferKeyPrefix   = "PUBLISH_ENGINE::BUFFER::"
	EventStatusPrefix = "EVENT::STATUS::"
	BufferTTL         = 120 * time.Second
	EventStatusTTL    = 120 * time.Second
)

// BackendRedis is used for stream_tech API buffer and event status (Section 7.4).
type BackendRedis struct {
	Client *redis.Client
	cfg    *config.Config
}

// NewBackendRedis creates a client for backend Redis (buffer/event status).
func NewBackendRedis(cfg *config.Config) (*BackendRedis, error) {
	addr := cfg.BackendRedisAddr()
	opts := &redis.Options{
		Addr:     addr,
		Password: cfg.RedisPassword,
		DB:       0,
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("backend redis ping: %w", err)
	}
	return &BackendRedis{Client: client, cfg: cfg}, nil
}

// StoreDataInRedis stores payload in buffer hash: key = PUBLISH_ENGINE::BUFFER::{routing_key}, field = eventName, value = JSON.
func (b *BackendRedis) StoreDataInRedis(ctx context.Context, routingKey, eventName string, payload interface{}) error {
	key := BufferKeyPrefix + routingKey
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if eventName == "" {
		eventName = "payload"
	}
	pipe := b.Client.Pipeline()
	pipe.HSet(ctx, key, eventName, string(payloadJSON))
	pipe.Expire(ctx, key, BufferTTL)
	_, err = pipe.Exec(ctx)
	return err
}

// IncrementAndCheckCompletion increments __processed_count__ and if >= total_messages sets EVENT::STATUS::{routing_key}.
func (b *BackendRedis) IncrementAndCheckCompletion(ctx context.Context, routingKey string) error {
	key := BufferKeyPrefix + routingKey
	n, err := b.Client.HIncrBy(ctx, key, "__processed_count__", 1).Result()
	if err != nil {
		return err
	}
	meta, err := b.Client.HGet(ctx, key, "__meta__").Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}
	var metaObj struct {
		TotalMessages int `json:"total_messages"`
	}
	if err := json.Unmarshal([]byte(meta), &metaObj); err != nil {
		return nil
	}
	if int(n) >= metaObj.TotalMessages {
		statusKey := EventStatusPrefix + routingKey
		if err := b.Client.Set(ctx, statusKey, "BUFFERED_EVENT_IN_GO_DISPATCHER_TO_REDIS", EventStatusTTL).Err(); err != nil {
			return err
		}
	}
	return nil
}

// UpdateEventStatus sets EVENT::STATUS::{routing_key} to status.
func (b *BackendRedis) UpdateEventStatus(ctx context.Context, routingKey, status string) error {
	key := EventStatusPrefix + routingKey
	return b.Client.Set(ctx, key, status, 20*time.Second).Err()
}

// Close closes the connection.
func (b *BackendRedis) Close() error {
	return b.Client.Close()
}
