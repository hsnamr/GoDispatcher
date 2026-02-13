package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/halamri/go-dispatcher/internal/config"
	"github.com/halamri/go-dispatcher/internal/models"
	"github.com/halamri/go-dispatcher/internal/redis"
)

// Worker consumes from the sender queue and publishes to Redis streams, Redis buffer, or live service.
type Worker struct {
	cfg     *config.Config
	broker  *redis.Broker
	backend *redis.BackendRedis
	queue   <-chan *models.SenderQueueItem
	log     *slog.Logger
}

// NewWorker creates a publisher worker.
func NewWorker(cfg *config.Config, broker *redis.Broker, backend *redis.BackendRedis, queue <-chan *models.SenderQueueItem, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{cfg: cfg, broker: broker, backend: backend, queue: queue, log: logger}
}

// Run runs the publisher loop until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-w.queue:
			if !ok {
				return
			}
			w.handleItem(ctx, item)
		}
	}
}

func (w *Worker) handleItem(ctx context.Context, item *models.SenderQueueItem) {
	if item == nil || item.Data == nil {
		return
	}
	data := item.Data

	// Completion marker for stream_tech API
	if data.CompletionMarker {
		if err := w.backend.IncrementAndCheckCompletion(ctx, data.RoutingKey); err != nil {
			w.log.Error("increment_and_check_completion failed", "routing_key", data.RoutingKey, "error", err)
		}
		return
	}

	msg := data.Message
	if msg == nil {
		return
	}

	routingKey := data.RoutingKey
	if w.cfg.OverrideOutput && w.cfg.OverrideRoutingKey != "" {
		routingKey = w.cfg.OverrideRoutingKey
	}

	// stream_tech SSE: POST to live service
	if msg.StreamTech == "SSE" {
		w.sendSSEEvent(ctx, data)
		return
	}

	// stream_tech API: store in Redis buffer only
	if msg.StreamTech == "API" {
		if err := w.backend.StoreDataInRedis(ctx, routingKey, msg.EventName, msg); err != nil {
			w.log.Error("store_data_in_redis failed", "routing_key", routingKey, "error", err)
		}
		return
	}

	// Default: publish to Redis output stream
	w.publishToStream(ctx, routingKey, msg)
}

func (w *Worker) sendSSEEvent(ctx context.Context, data *models.SenderQueueItemData) {
	msg := data.Message
	url := w.cfg.LiveServiceURL + "/publisher/event"
	if w.cfg.LiveServiceURL == "" {
		w.log.Warn("APP_LIVE_SERVICE_URL not set, skipping SSE send")
		return
	}
	body := map[string]interface{}{
		"eventName":   msg.EventName,
		"eventData":   msg.EventData,
		"queue_name":  msg.QueueName,
		"routing_key": msg.RoutingKey,
	}
	if msg.MonitorID != "" {
		body["monitor_id"] = msg.MonitorID
	}
	if msg.DataSource != "" {
		body["data_source"] = msg.DataSource
	}
	jsonBody, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		w.log.Error("create SSE request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	timeout := w.cfg.AIAPIsTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		w.log.Error("SSE POST failed", "url", url, "error", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.log.Error("SSE POST non-2xx", "url", url, "status", resp.StatusCode)
	}
}

func (w *Worker) publishToStream(ctx context.Context, routingKey string, msg *models.OutgoingMessage) {
	stream := w.broker.OutputStreamName(routingKey)
	payload, err := json.Marshal(msg)
	if err != nil {
		w.log.Error("marshal message failed", "error", err)
		return
	}
	values := map[string]interface{}{
		"payload":     string(payload),
		"eventName":   msg.EventName,
		"routing_key": routingKey,
	}
	_, err = w.broker.AddToStream(ctx, stream, values)
	if err != nil {
		w.log.Error("XADD failed", "stream", stream, "error", err)
		// Optional: reconnect and retry once (spec 7.6)
		return
	}
}

// PrepareDataToFrontend builds an OutgoingMessage envelope from widget event name and data (spec 7.2).
func PrepareDataToFrontend(eventName string, eventData interface{}, routingKey, queueName string, params *models.EventData, cfg *config.Config) *models.OutgoingMessage {
	msg := &models.OutgoingMessage{
		EventName:   eventName,
		EventData:   eventData,
		QueueName:   queueName,
		StreamTech:  params.StreamTech,
		MonitorID:   params.MonitorID,
		DataSource:  params.DataSource,
		Language:    params.Language,
	}
	if cfg != nil {
		msg.ProjectNameEN = cfg.WhitelabelProjectNameEN
		msg.AIProductNameEN = cfg.WhitelabelAIProductNameEN
		msg.ProjectNameAR = cfg.WhitelabelProjectNameAR
		msg.AIProductNameAR = cfg.WhitelabelAIProductNameAR
		msg.PoweredBy = cfg.WhitelabelPoweredBy
		msg.LegalName = cfg.WhitelabelLegalName
		msg.PoweredBySentence = cfg.WhitelabelPoweredBySentence
		msg.DisclaimerAR = cfg.WhitelabelDisclaimerAR
		msg.ContactEmail = cfg.WhitelabelContactEmail
		msg.AIProductPPTLogo = cfg.WhitelabelAIProductPPTLogo
		if cfg.WhitelabelAssumeResponsibility {
			msg.AssumeResponsibility = &cfg.WhitelabelAssumeResponsibility
		}
	}
	return msg
}
