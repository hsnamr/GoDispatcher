package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/halamri/go-dispatcher/internal/config"
	"github.com/halamri/go-dispatcher/internal/controller"
	"github.com/halamri/go-dispatcher/internal/models"
	"github.com/halamri/go-dispatcher/internal/redis"
)

const (
	readBlockMs    = 2000
	senderQueueCap = 1000
)

// CalcExportProducer sends export requests to the calc data_fetcher topic (e.g. Kafka).
type CalcExportProducer interface {
	SendDataFetcher(ctx context.Context, incoming *models.IncomingMessage) error
}

// Consumer consumes from Redis stream or Kafka and dispatches to controller + dispatcher.
type Consumer struct {
	cfg               *config.Config
	broker            *redis.Broker
	backend           *redis.BackendRedis
	ctrl              *controller.Controller
	senderCh          chan *models.SenderQueueItem
	log               *slog.Logger
	calcExportProducer CalcExportProducer // optional; when set, export path produces to staci.calc.data_fetcher
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

// SetCalcExportProducer sets the optional producer for export path (staci.calc.data_fetcher).
func (c *Consumer) SetCalcExportProducer(p CalcExportProducer) {
	c.calcExportProducer = p
}

// SenderQueue returns the channel the dispatcher worker should read from.
func (c *Consumer) SenderQueue() chan *models.SenderQueueItem {
	return c.senderCh
}

// Run starts the Redis consumer loop. It creates the consumer group, then reads messages and processes them.
// Use Kafka consumer (RunConsumer) when cfg.UseKafka() is true.
func (c *Consumer) Run(ctx context.Context) error {
	if c.broker == nil {
		return nil
	}
	stream := c.cfg.DispatcherInputStream
	group := c.cfg.DispatcherConsumerGroup
	consumerName := c.cfg.DispatcherConsumerName

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

		ack, reason := c.ProcessMessage(ctx, &incoming)
		if ack {
			c.broker.Ack(ctx, stream, group, messageID)
		} else {
			c.deadLetter(ctx, stream, group, messageID, &incoming, reason)
		}
	}
}

// ProcessMessage processes one incoming message. Returns (true, "") to ack, (false, reason) to deadLetter.
// Used by both Redis and Kafka consumers.
func (c *Consumer) ProcessMessage(ctx context.Context, incoming *models.IncomingMessage) (ack bool, deadLetterReason string) {
	eventName := incoming.EventName
	eventData := incoming.EventData

	if eventData == nil {
		return true, ""
	}

	// Accepted events only
	if !models.AcceptedEventNames[eventName] {
		return true, ""
	}

	// Export path: produce to staci.calc.data_fetcher and ack (instead of skip)
	if eventData.ExporterID != "" && c.calcExportProducer != nil {
		isExportRequest := eventData.ExportXLSX ||
			eventName == "staci.calc.export_request" ||
			eventName == "fedi_listener_go_dispatcher" ||
			eventName == "calc_export"
		if isExportRequest {
			if err := c.calcExportProducer.SendDataFetcher(ctx, incoming); err != nil {
				c.log.Error("send data fetcher failed", "error", err, "exporter_id", eventData.ExporterID)
				return false, "export_send_failed"
			}
			return true, ""
		}
	}

	// Conditional skip: different pipeline (spec 5.3)
	if eventData.ExporterID != "" || eventData.JobID != "" || eventData.IsLiveDashboard || eventData.IsNewsletterReport || eventData.IsExportEngagements {
		return true, ""
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
			return false, "stale_request"
		}
	}

	// Special handling: fedi_listener_go_dispatcher with export_xlsx and timeout (spec 5.4)
	const messageDispatcherTimeoutSec = 7200
	if (eventName == "fedi_listener_go_dispatcher" || eventName == "staci.fedi.dispatcher") && eventData.ExportXLSX && eventData.RequestTimestamp > 0 {
		if time.Now().Unix()-eventData.RequestTimestamp > messageDispatcherTimeoutSec {
			return false, "export_xlsx_timeout"
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
		return false, "controller_error"
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

	return true, ""
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
	_, _ = c.broker.AddToStream(ctx, c.cfg.DispatcherDeadletterStream, values)
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
