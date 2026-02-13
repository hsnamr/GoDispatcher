package strategies

import (
	"context"

	"github.com/halamri/go-dispatcher/internal/controller"
	"github.com/halamri/go-dispatcher/internal/models"
)

// StubStrategy is a minimal strategy that pushes one placeholder widget message.
type StubStrategy struct{}

// CallPageData pushes a single placeholder payload to the sender queue.
func (s *StubStrategy) CallPageData(ctx context.Context, params *models.EventData, senderQueue chan<- *models.SenderQueueItem) error {
	eventName := "Stub__Widget"
	if params.PageName != "" {
		eventName = params.PageName + "__stub"
	}
	item := &models.SenderQueueItem{
		Data: &models.SenderQueueItemData{
			Message: &models.OutgoingMessage{
				EventName: eventName,
				EventData: map[string]interface{}{
					"placeholder": true,
					"page":       params.PageName,
					"routing_key": params.RoutingKey,
				},
				QueueName:  params.QueueName,
				StreamTech: params.StreamTech,
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

// RegisterStubStrategies registers stub strategies for all spec-referenced page names so every event has a handler.
func RegisterStubStrategies(r *controller.Registry) {
	stub := &StubStrategy{}

	pages := []string{
		"CustomDashboard", "AccountPage", "EngagementsPage", "CustomerCarePage", "PostsPage",
		"CommentsMentionsPage", "AuthorsPage", "QuestionsPage", "ResearchInteractionsPage", "ResearchInsightsPage",
		"ResearchChatsPage", "ResearchChatsInteractionsPage", "ResearchSingleDataSourceInsightsPage", "ResearchSingleDataSourceInteractionsPage",
		"CDPProfileInteractionsPage", "SurveyPage", "SurveyIndividualQuestionsInsightsPage",
		"EngagementAnalyticsPage", "EngagementsNotesPage", "EngagementsSingleInteractionDMHistoryPage",
		"EngagementSurveyFeedbackPage", "SingleInteractionPage", "CsatQuestionsResponsesCountPage",
		"CsatQuestionResponsesPage", "CsatUserResponsesPage",
		"UnifiedAccountPerformancePage", "UnifiedAccountContentPage", "UnifiedVisualContentPage",
		"UnifiedEngagementInsightsPage", "UnifiedCustomerCarePerformancePage", "UnifiedPostPage",
		"UnifiedQuestionsPage", "UnifiedAuthorsPage", "UnifiedProfilePage", "UnifiedDynamicWidgetsPage",
		"UnifiedBenchmarkPage", "EngagementsProductPage", "EngagementAnalyticsPageExport",
		"Default",
	}
	for _, p := range pages {
		r.Register("", p, "", stub)
	}
	// Dashboards
	r.Register("Dashboards", "CustomDashboard", "", stub)
	r.Register("Dashboards", "AccountPage", "", stub)
	r.Register("Dashboards", "EngagementsPage", "", stub)
	r.Register("Dashboards", "CustomerCarePage", "", stub)
	r.Register("Dashboards", "PostsPage", "", stub)
	r.Register("Dashboards", "CommentsMentionsPage", "", stub)
	r.Register("Dashboards", "AuthorsPage", "", stub)
	r.Register("Dashboards", "QuestionsPage", "", stub)
	// Alerts (strategy keyed by page_alert_notification)
	alerts := []string{"TrendingAlert", "NegativePostsAlert", "NewPostAlert", "HighVolumeReachAlert", "ViralTweetAlert", "InfluencerAlert", "VerifiedAuthorAlert", "HourlyAlertNotification"}
	for _, a := range alerts {
		r.Register("Alerts", a, "", stub)
	}
	// PUBLIC_API
	r.Register("PUBLIC_API", "Account", "", stub)
	r.Register("PUBLIC_API", "Engagements", "", stub)
	r.Register("PUBLIC_API", "CustomerCare", "", stub)
	r.Register("PUBLIC_API", "Posts", "", stub)
	r.Register("PUBLIC_API", "Analytics", "", stub)
	r.Register("PUBLIC_API", "Chats", "", stub)
	r.Register("PUBLIC_API", "Interactions", "", stub)
	r.Register("PUBLIC_API", "SingleSource", "", stub)
	r.Register("PUBLIC_API", "SocialMedia", "", stub)
	r.Register("PUBLIC_API", "EngagementAnalytics", "", stub)
}
