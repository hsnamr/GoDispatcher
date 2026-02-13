package models

// IncomingMessage is the JSON structure consumed from Redis Streams.
type IncomingMessage struct {
	EventName        string     `json:"eventName"`
	EventData        *EventData `json:"eventData"`
	SelectedWidgets  []string   `json:"selected_widgets,omitempty"`
}

// EventData is the payload inside eventData.
type EventData struct {
	QueueName       string                 `json:"queue_name"`
	RoutingKey      string                 `json:"routing_key"`
	RequestTimestamp int64                 `json:"request_timestamp,omitempty"`
	PageName        string                 `json:"page_name,omitempty"`
	CompanySchema   string                 `json:"company_schema,omitempty"`
	MonitorID       string                 `json:"monitor_id,omitempty"`
	StartDate       string                 `json:"start_date,omitempty"`
	EndDate         string                 `json:"end_date,omitempty"`
	Filters         interface{}            `json:"filters,omitempty"`
	Language        string                 `json:"language,omitempty"`
	DataSource      string                 `json:"data_source,omitempty"`
	DataSourceName  string                 `json:"data_source_name,omitempty"`
	Metrics         interface{}            `json:"metrics,omitempty"`
	SelectedWidgets []string               `json:"selected_widgets,omitempty"`
	StreamTech      string                 `json:"stream_tech,omitempty"`
	CompanyID       string                 `json:"company_id,omitempty"`
	Product         string                 `json:"product,omitempty"`
	PageAlertNotification string           `json:"page_alert_notification,omitempty"`
	ExportXLSX      bool                   `json:"export_xlsx,omitempty"`
	ExporterID      string                 `json:"exporter_id,omitempty"`
	JobID           string                 `json:"job_id,omitempty"`
	IsLiveDashboard bool                   `json:"is_live_dashboard,omitempty"`
	IsNewsletterReport bool                `json:"is_newsletter_report,omitempty"`
	IsExportEngagements bool               `json:"is_export_engagements,omitempty"`
	Extra           map[string]interface{} `json:"-"` // rest of fields stored here when unmarshalling
}

// OutgoingMessage is the envelope sent to frontend (data.message in spec).
type OutgoingMessage struct {
	EventName   string                 `json:"eventName"`
	EventData   interface{}            `json:"eventData"`
	MonitorID   string                 `json:"monitor_id,omitempty"`
	DataSource  string                 `json:"data_source,omitempty"`
	Metadata    interface{}            `json:"metadata,omitempty"`
	StreamTech  string                 `json:"stream_tech,omitempty"`
	QueueName   string                 `json:"queue_name,omitempty"`
	ExporterID  string                 `json:"exporter_id,omitempty"`
	JobID       string                 `json:"job_id,omitempty"`
	Language    string                 `json:"language,omitempty"`
	ProjectNameEN string               `json:"project_name_en,omitempty"`
	AIProductNameEN string             `json:"ai_product_name_en,omitempty"`
	ProjectNameAR string               `json:"project_name_ar,omitempty"`
	AIProductNameAR string             `json:"ai_product_name_ar,omitempty"`
	PoweredBy   string                 `json:"powered_by,omitempty"`
	LegalName   string                 `json:"legal_name,omitempty"`
	PoweredBySentence string           `json:"powered_by_sentence,omitempty"`
	DisclaimerAR string               `json:"disclaimer_ar,omitempty"`
	ContactEmail string               `json:"contact_email,omitempty"`
	AssumeResponsibility *bool         `json:"assume_responsibility,omitempty"`
	AIProductPPTLogo string           `json:"ai_product_ppt_logo,omitempty"`
}

// SenderQueueItem is what strategies push onto the sender queue (metadata, carrier, data).
type SenderQueueItem struct {
	Metadata map[string]string      // optional message hash for tracing
	Carrier  map[string]string      // trace context
	Data     *SenderQueueItemData   // required
}

// SenderQueueItemData holds the payload and routing for the dispatcher.
type SenderQueueItemData struct {
	Message    *OutgoingMessage     `json:"message"`
	RoutingKey string               `json:"routing_key"`
	Exchange   string               `json:"exchange,omitempty"`
	// Dynamic dispatch flags (from spec 7.1)
	ExporterID         string `json:"exporter_id,omitempty"`
	AlertID            string `json:"alert_id,omitempty"`
	StaciActive        bool   `json:"staci_active,omitempty"`
	JobID              string `json:"job_id,omitempty"`
	IsLiveDashboard    bool   `json:"is_live_dashboard,omitempty"`
	IsNewsletterReport bool   `json:"is_newsletter_report,omitempty"`
	IsExportEngagements bool  `json:"is_export_engagements,omitempty"`
	// Completion marker for stream_tech API
	CompletionMarker bool   `json:"__completion_marker__,omitempty"`
}
