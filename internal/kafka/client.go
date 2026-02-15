package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/halamri/go-dispatcher/internal/config"
	"github.com/halamri/go-dispatcher/internal/models"
	"github.com/segmentio/kafka-go"
)

// Client provides Kafka producer and consumer for Staci canonical topics.
type Client struct {
	cfg     *config.Config
	reader  *kafka.Reader
	writer  *kafka.Writer
	dlqWr   *kafka.Writer
	fetcher *kafka.Writer // for staci.calc.data_fetcher
}

// NewClient creates a Kafka client for the dispatcher (reader on requests, writer on output and optional DLQ/fetcher).
func NewClient(cfg *config.Config) (*Client, error) {
	brokers := strings.Split(strings.TrimSpace(cfg.KafkaBrokers), ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          cfg.KafkaTopicDispatcherRequests,
		GroupID:        cfg.KafkaConsumerGroupDispatcher,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        2 * time.Second,
		StartOffset:    kafka.FirstOffset,
		CommitInterval: 0,
	})
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        cfg.KafkaTopicDispatcherOutput,
		Balancer:     &kafka.Hash{},
		BatchTimeout: 10 * time.Millisecond,
	}
	dlqWr := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    cfg.KafkaTopicDispatcherRequests + ".dlq",
		Balancer: &kafka.Hash{},
	}
	fetcher := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    cfg.KafkaTopicCalcDataFetcher,
		Balancer: &kafka.Hash{},
	}
	return &Client{
		cfg:     cfg,
		reader:  reader,
		writer:  writer,
		dlqWr:   dlqWr,
		fetcher: fetcher,
	}, nil
}

// Close closes the Kafka client.
func (c *Client) Close() error {
	_ = c.reader.Close()
	_ = c.writer.Close()
	_ = c.dlqWr.Close()
	_ = c.fetcher.Close()
	return nil
}

// SendDataFetcher produces a staci.calc.ods_data_fetcher message to staci.calc.data_fetcher (for export path).
func (c *Client) SendDataFetcher(ctx context.Context, incoming *models.IncomingMessage) error {
	if incoming.EventData == nil {
		return nil
	}
	ed := incoming.EventData
	payload := map[string]interface{}{
		"eventName": "staci.calc.ods_data_fetcher",
		"eventData": map[string]interface{}{
			"data": map[string]interface{}{
				"exporter_id":   ed.ExporterID,
				"page_name":    ed.PageName,
				"name":         "export",
				"company_schema": ed.CompanySchema,
				"filters":      ed.Filters,
				"monitor_id":   ed.MonitorID,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	key := ed.ExporterID
	if key == "" {
		key = "export"
	}
	return c.fetcher.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: body,
	})
}

// ProduceOutput writes an outgoing message to staci.dispatcher.output.
func (c *Client) ProduceOutput(ctx context.Context, routingKey string, payload []byte) error {
	return c.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(routingKey),
		Value: payload,
	})
}

// DeadLetter sends the payload to the DLQ topic (staci.dispatcher.requests.dlq).
func (c *Client) DeadLetter(ctx context.Context, key string, payload []byte, reason string) error {
	msg := payload
	if reason != "" {
		// Wrap in envelope with reason
		wrapped := map[string]interface{}{
			"payload": string(payload),
			"reason":  reason,
		}
		if b, err := json.Marshal(wrapped); err == nil {
			msg = b
		}
	}
	return c.dlqWr.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: msg,
	})
}

// RunConsumer reads from staci.dispatcher.requests and calls process for each message.
// If process returns (false, reason), the message is sent to DLQ and offset is committed.
// If process returns (true, _), offset is committed.
func (c *Client) RunConsumer(ctx context.Context, log *slog.Logger, process func(ctx context.Context, incoming *models.IncomingMessage) (ack bool, deadLetterReason string)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Error("kafka fetch failed", "error", err)
			continue
		}
		var incoming models.IncomingMessage
		if err := json.Unmarshal(msg.Value, &incoming); err != nil {
			log.Error("invalid JSON payload", "error", err)
			_ = c.reader.CommitMessages(ctx, msg)
			continue
		}
		ack, reason := process(ctx, &incoming)
		if !ack {
			_ = c.DeadLetter(ctx, string(msg.Key), msg.Value, reason)
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Error("kafka commit failed", "error", err)
		}
	}
}
