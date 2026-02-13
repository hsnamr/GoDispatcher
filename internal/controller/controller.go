package controller

import (
	"context"
	"sync"

	"github.com/halamri/go-dispatcher/internal/models"
)

// Strategy is the interface for page strategies (call_page_data).
type Strategy interface {
	CallPageData(ctx context.Context, params *models.EventData, senderQueue chan<- *models.SenderQueueItem) error
}

// Registry maps (product, page_name, page_alert_notification) to a strategy.
type Registry struct {
	mu        sync.RWMutex
	strategies map[string]Strategy // key: product|page_name|page_alert_notification
}

// NewRegistry returns a new strategy registry.
func NewRegistry() *Registry {
	return &Registry{strategies: make(map[string]Strategy)}
}

// Register adds a strategy for the given product, pageName, and optional pageAlertNotification.
func (r *Registry) Register(product, pageName, pageAlertNotification string, s Strategy) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strategyKey(product, pageName, pageAlertNotification)
	r.strategies[key] = s
}

// Get returns the strategy for the given keys, or nil.
func (r *Registry) Get(product, pageName, pageAlertNotification string) Strategy {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Try exact match first
	key := strategyKey(product, pageName, pageAlertNotification)
	if s, ok := r.strategies[key]; ok {
		return s
	}
	// Fallback: product empty, page_name only
	key = strategyKey("", pageName, "")
	if s, ok := r.strategies[key]; ok {
		return s
	}
	return nil
}

func strategyKey(product, pageName, pageAlertNotification string) string {
	if pageAlertNotification != "" {
		return product + "|" + pageName + "|" + pageAlertNotification
	}
	if product != "" {
		return product + "|" + pageName
	}
	return pageName
}

// Controller runs the selected strategy and feeds the sender queue.
type Controller struct {
	Registry *Registry
}

// NewController creates a controller with the given registry.
func NewController(registry *Registry) *Controller {
	return &Controller{Registry: registry}
}

// Run selects strategy by params.Product and params.PageName (and params.PageAlertNotification for Alerts),
// then calls CallPageData. If no strategy is found, it pushes a single placeholder message so the pipeline completes.
func (c *Controller) Run(ctx context.Context, params *models.EventData, senderQueue chan<- *models.SenderQueueItem) error {
	if params == nil {
		return nil
	}
	product := params.Product
	pageName := params.PageName
	pageAlertNotification := params.PageAlertNotification
	if product == "Alerts" && pageAlertNotification != "" {
		pageName = pageAlertNotification // spec: Alerts → {PageAlertNotification}Strategy
	}
	if pageName == "" {
		pageName = "Default"
	}
	strategy := c.Registry.Get(product, pageName, pageAlertNotification)
	if strategy != nil {
		return strategy.CallPageData(ctx, params, senderQueue)
	}
	// No strategy: push a minimal placeholder so dispatcher can ack and complete
	item := &models.SenderQueueItem{
		Data: &models.SenderQueueItemData{
			Message: &models.OutgoingMessage{
				EventName: "Dispatcher__Placeholder",
				EventData: map[string]string{"status": "no_strategy", "page": pageName},
			},
			RoutingKey: params.RoutingKey,
		},
	}
	select {
	case senderQueue <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
