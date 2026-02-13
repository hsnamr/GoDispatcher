package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/halamri/go-dispatcher/internal/config"
)

// Broker wraps Redis client for streams and general I/O.
type Broker struct {
	Client *redis.Client
	cfg    *config.Config
}

// NewBroker creates a Redis client from config.
func NewBroker(cfg *config.Config) (*Broker, error) {
	opts := &redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       0,
	}
	if cfg.RedisTLS {
		opts.TLSConfig = nil // caller can set TLS config if needed
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Broker{Client: client, cfg: cfg}, nil
}

// EnsureConsumerGroup creates the consumer group on the input stream if it does not exist.
func (b *Broker) EnsureConsumerGroup(ctx context.Context, stream, group, startID string) error {
	if startID == "" {
		startID = "$"
	}
	err := b.Client.XGroupCreateMkStream(ctx, stream, group, startID).Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// ReadGroup blocks and reads one message from the stream (XREADGROUP GROUP ... COUNT 1 BLOCK ms STREAMS stream >).
func (b *Broker) ReadGroup(ctx context.Context, stream, group, consumer string, blockMs time.Duration) (streamKey string, messageID string, fields map[string]interface{}, err error) {
	streams, err := b.Client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    blockMs,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", "", nil, nil // no message
		}
		return "", "", nil, err
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return "", "", nil, nil
	}
	msg := streams[0].Messages[0]
	m := make(map[string]interface{})
	for k, v := range msg.Values {
		m[k] = v
	}
	return streams[0].Stream, msg.ID, m, nil
}

// Ack acknowledges a message.
func (b *Broker) Ack(ctx context.Context, stream, group string, ids ...string) error {
	return b.Client.XAck(ctx, stream, group, ids...).Err()
}

// AddToStream adds an entry to a stream (e.g. output or dead-letter) with optional MAXLEN.
func (b *Broker) AddToStream(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	args := &redis.XAddArgs{Stream: stream, Values: values}
	if b.cfg.DispatcherStreamMaxLen > 0 {
		args.MaxLenApprox = b.cfg.DispatcherStreamMaxLen
	}
	return b.Client.XAdd(ctx, args).Result()
}

// OutputStreamName returns the full stream key for a routing key.
func (b *Broker) OutputStreamName(routingKey string) string {
	if b.cfg.OverrideOutput && b.cfg.OverrideRoutingKey != "" {
		routingKey = b.cfg.OverrideRoutingKey
	}
	return b.cfg.DispatcherOutputStreamPrefix + routingKey
}

// Close closes the Redis connection.
func (b *Broker) Close() error {
	return b.Client.Close()
}
