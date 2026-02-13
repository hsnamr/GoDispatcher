package models

// AcceptedEventNames is the set of event names the dispatcher processes (spec 5.2).
var AcceptedEventNames = map[string]bool{
	"fedi_listener_go_dispatcher":                        true,
	"interactions":                                       true,
	"insights":                                           true,
	"cdp_profile_interactions":                           true,
	"aggregate_dashboard_go_dispatcher":                   true,
	"custom_dashboard_go_dispatcher":                      true,
	"fedi_data":                                          true,
	"alert_data":                                         true,
	"chats_page_data":                                    true,
	"chats_interactions":                                 true,
	"survey_insights_go_dispatcher":                      true,
	"survey_individual_questions_insights_go_dispatcher":  true,
	"benchmark_go_dispatcher":                             true,
	"fedi_private_data":                                  true,
	"fedi_interactions":                                  true,
	"fedi_private_interactions":                          true,
	"custom_projects":                                    true,
	"explore_go_dispatcher":                              true,
	"engagements_go_dispatcher":                          true,
	"engagements_notes_go_dispatcher":                     true,
	"engagements_analytics_go_dispatcher":                 true,
	"engagements_single_interaction_dm_history":          true,
	"single_post_go_dispatcher":                           true,
	"engagements_survey_feedback_go_dispatcher":          true,
	"top_trends_go_dispatcher":                            true,
	"single_interaction_go_dispatcher":                    true,
	"csat_questions_responses_count":                      true,
	"csat_question_responses":                             true,
	"csat_user_responses":                                 true,
	"calc_export":                                        true,
	"unified_json":                                       true,
	"qdtranscriber_go_dispatcher":                        true,
}

// EngagementsPageNameMapping maps eventData.page_name for engagements_go_dispatcher (spec 6.3).
var EngagementsPageNameMapping = map[string]string{
	"engagements":                "EngagementsProductPage",
	"engagements_profile_data":   "EngagementsProfileDataPage",
	"engagements_logs_data":      "EngagementsSlasPage",
	"engagements_notes":          "EngagementsNotesPage",
	"engagements_analytics":      "EngagementAnalyticsPage",
	"engagements_single_interaction_dm_history": "EngagementsSingleInteractionDMHistoryPage",
	"engagement_survey_feedback":  "EngagementSurveyFeedbackPage",
}

// UnifiedJSONPageMapping maps eventData.page_name to strategy page name for eventName unified_json (spec 6.4).
var UnifiedJSONPageMapping = map[string]string{
	"account_performance_page": "UnifiedAccountPerformancePage",
	"account_content_page":     "UnifiedAccountContentPage",
	"visual_content_page":      "UnifiedVisualContentPage",
	"engagement_insights_page": "UnifiedEngagementInsightsPage",
	"customer_care_page":       "UnifiedCustomerCarePerformancePage",
	"posts_page":               "UnifiedPostPage",
	"engagements_page":         "UnifiedPostPage",
	"questions_page":           "UnifiedQuestionsPage",
	"authors_page":             "UnifiedAuthorsPage",
	"profile_page":             "UnifiedProfilePage",
	"dynamic_widgets_page":     "UnifiedDynamicWidgetsPage",
	"benchmark_page":           "UnifiedBenchmarkPage",
}

// EventToPageEnrichment maps eventName to page_name (and product/data_source_name where specified) - spec 6.3.
func EventToPageEnrichment(eventName string, eventData *EventData) {
	if eventData == nil {
		return
	}
	switch eventName {
	case "interactions":
		eventData.PageName = "ResearchInteractionsPage"
	case "insights":
		eventData.PageName = "ResearchInsightsPage"
	case "chats_page_data":
		eventData.PageName = "ResearchChatsPage"
	case "chats_interactions":
		eventData.PageName = "ResearchChatsInteractionsPage"
	case "fedi_private_data":
		eventData.PageName = "ResearchSingleDataSourceInsightsPage"
		eventData.DataSourceName = "Fediprivate"
	case "fedi_private_interactions":
		eventData.PageName = "ResearchSingleDataSourceInteractionsPage"
		eventData.DataSourceName = "Fediprivate"
	case "aggregate_dashboard_go_dispatcher", "custom_dashboard_go_dispatcher":
		eventData.Product = "Dashboards"
	case "cdp_profile_interactions":
		eventData.PageName = "CDPProfileInteractionsPage"
	case "survey_insights_go_dispatcher":
		eventData.PageName = "SurveyPage"
	case "survey_individual_questions_insights_go_dispatcher":
		eventData.PageName = "SurveyIndividualQuestionsInsightsPage"
	case "engagements_notes_go_dispatcher":
		eventData.PageName = "EngagementsNotesPage"
	case "engagements_analytics_go_dispatcher":
		eventData.PageName = "EngagementAnalyticsPage"
	case "engagements_go_dispatcher":
		if eventData.PageName != "" {
			if mapped, ok := EngagementsPageNameMapping[eventData.PageName]; ok {
				eventData.PageName = mapped
			}
		} else {
			eventData.PageName = "EngagementsProductPage"
		}
	case "engagements_single_interaction_dm_history":
		eventData.PageName = "EngagementsSingleInteractionDMHistoryPage"
	case "engagements_survey_feedback_go_dispatcher":
		eventData.PageName = "EngagementSurveyFeedbackPage"
	case "single_interaction_go_dispatcher":
		eventData.PageName = "SingleInteractionPage"
	case "csat_questions_responses_count":
		eventData.PageName = "FulfillmentQuestionsResponsesCountPage"
	case "csat_question_responses":
		eventData.PageName = "FulfillmentQuestionResponsesPage"
	case "csat_user_responses":
		eventData.PageName = "FulfillmentUserResponsesPage"
	case "unified_json":
		if eventData.PageName != "" {
			if mapped, ok := UnifiedJSONPageMapping[eventData.PageName]; ok {
				eventData.PageName = mapped
			}
		}
	default:
		// Other events may already have page_name in eventData (e.g. engagements_go_dispatcher)
	}
}
