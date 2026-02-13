package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/halamri/go-dispatcher/internal/config"
	"github.com/halamri/go-dispatcher/internal/controller"
	"github.com/halamri/go-dispatcher/internal/models"
	"github.com/halamri/go-dispatcher/internal/publisher"
	"github.com/halamri/go-dispatcher/internal/redis"
)

const (
	readBlockMs    = 2000
	senderQueueCap = 1000
)

// Consumer consumes from Redis stream and dispatches to controller + publisher.
type Consumer struct {
	cfg       *config.Config
	broker    *redis.Broker
	backend   *redis.BackendRedis
	ctrl      *controller.Controller
	senderCh  chan *models.SenderQueueItem
	log       *slog.Logger
}

// NewConsumer creates a consumer.
func NewConsumer(cfg *config.Config, broker *redis.Broker, backend *redis.BackendRedis, ctrl *controller.Controller, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{
		cfg:      cfg,
		broker:   broker,
		backend:  backend,
		ctrl:     ctrl,
		senderCh: make(chan *models.SenderQueueItem, senderQueueCap),
		log:      logger,
	}
}

// SenderQueue returns the channel the publisher worker should read from.
func (c *Consumer) SenderQueue() chan *models.SenderQueueItem {
	return c.senderCh
}

// Run starts the consumer loop. It creates the consumer group, then reads messages and processes them.
func (c *Consumer) Run(ctx context.Context) error {
	stream := c.cfg.PublisherInputStream
	group := c.cfg.PublisherConsumerGroup
	consumerName := c.cfg.PublisherConsumerName

	if err := c.broker.EnsureConsumerGroup(ctx, stream, group, "$"); err != nil {
		return err
	}

	block := readBlockMs * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		streamKey, messageID, fields, err := c.broker.ReadGroup(ctx, stream, group, consumerName, block)
		if err != nil {
			c.log.Error("XREADGROUP failed", "error", err)
			continue
		}
		if streamKey == "" || messageID == "" {
			continue
		}

		payloadStr, _ := stringFromMap(fields, "payload")
		if payloadStr == "" {
			payloadStr, _ = stringFromMap(fields, "body")
		}
		if payloadStr == "" {
			c.broker.Ack(ctx, stream, group, messageID)
			continue
		}

		var incoming models.IncomingMessage
		if err := json.Unmarshal([]byte(payloadStr), &incoming); err != nil {
			c.log.Error("invalid JSON payload", "message_id", messageID, "error", err)
			c.broker.Ack(ctx, stream, group, messageID)
			continue
		}

		c.processMessage(ctx, stream, group, messageID, &incoming)
	}
}

func (c *Consumer) processMessage(ctx context.Context, stream, group, messageID string, incoming *models.IncomingMessage) {
	eventName := incoming.EventName
	eventData := incoming.EventData

	if eventData == nil {
		c.broker.Ack(ctx, stream, group, messageID)
		return
	}

	// Accepted events only
	if !models.AcceptedEventNames[eventName] {
		c.broker.Ack(ctx, stream, group, messageID)
		return
	}

	// Conditional skip: different pipeline (spec 5.3)
	if eventData.ExporterID != "" || eventData.JobID != "" || eventData.IsLiveDashboard || eventData.IsNewsletterReport || eventData.IsExportEngagements {
		c.broker.Ack(ctx, stream, group, messageID)
		return
	}
	if eventData.StreamTech != "" && eventData.StreamTech != "REDIS STREAM" {
		// Optional: handle API path; for main path we ack and skip or handle separately
	}

	// Merge selected_widgets from top-level
	if len(incoming.SelectedWidgets) > 0 {
		eventData.SelectedWidgets = append(eventData.SelectedWidgets, incoming.SelectedWidgets...)
	}

	// Stale request check
	if eventData.RequestTimestamp > 0 {
		elapsed := time.Now().Unix() - eventData.RequestTimestamp
		if elapsed > int64(c.cfg.EventTimeout.Seconds()) {
			c.deadLetter(ctx, stream, group, messageID, incoming, "stale_request")
			return
		}
	}

	// Special handling: fedi_listener_go_dispatcher with export_xlsx and timeout (spec 5.4)
	const messagePublisherTimeoutSec = 7200
	if eventName == "fedi_listener_go_dispatcher" && eventData.ExportXLSX && eventData.RequestTimestamp > 0 {
		if time.Now().Unix()-eventData.RequestTimestamp > messagePublisherTimeoutSec {
			c.deadLetter(ctx, stream, group, messageID, incoming, "export_xlsx_timeout")
			return
		}
	}

	// Enrich page_name / product from eventName
	models.EventToPageEnrichment(eventName, eventData)

	// Per-message timeout
	runCtx, cancel := context.WithTimeout(ctx, c.cfg.EventTimeout)
	defer cancel()

	// stream_tech API: set status to processing
	if eventData.StreamTech == "API" && eventData.RoutingKey != "" {
		_ = c.backend.UpdateEventStatus(runCtx, eventData.RoutingKey, "PROCESSING_EVENT_IN_PUBLISHER_CONTROLLER")
	}

	err := c.ctrl.Run(runCtx, eventData, c.senderCh)
	if err != nil {
		c.log.Error("controller run failed", "event_name", eventName, "routing_key", eventData.RoutingKey, "error", err)
		c.deadLetter(ctx, stream, group, messageID, incoming, "controller_error")
		return
	}

	// Completion marker for stream_tech API so frontend knows all widgets are done
	if eventData.StreamTech == "API" && eventData.RoutingKey != "" {
		select {
		case c.senderCh <- &models.SenderQueueItem{
			Data: &models.SenderQueueItemData{
				CompletionMarker: true,
				RoutingKey:       eventData.RoutingKey,
			},
		}:
		case <-runCtx.Done():
		}
	}

	c.broker.Ack(ctx, stream, group, messageID)
}

func (c *Consumer) deadLetter(ctx context.Context, stream, group, messageID string, incoming *models.IncomingMessage, reason string) {
	payload, _ := json.Marshal(incoming)
	values := map[string]interface{}{
		"payload":    string(payload),
		"eventName":  incoming.EventName,
		"reason":     reason,
		"message_id": messageID,
	}
	if incoming.EventData != nil {
		values["routing_key"] = incoming.EventData.RoutingKey
	}
	_, _ = c.broker.AddToStream(ctx, c.cfg.PublisherDeadletterStream, values)
	c.broker.Ack(ctx, stream, group, messageID)
}

func stringFromMap(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
