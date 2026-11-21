package models

import "time"

// AIPrediction represents an AI-generated ticket classification.
type AIPrediction struct {
	ID                 string    `json:"id"`
	TicketID           string    `json:"ticket_id"`
	ModelVersion       string    `json:"model_version"`
	Category           *string   `json:"category,omitempty"`
	PredictedPriority  *string   `json:"predicted_priority,omitempty"`
	Sentiment          *string   `json:"sentiment,omitempty"`
	Confidence         *float64  `json:"confidence,omitempty"`
	SuggestedTeamID    *string   `json:"suggested_team_id,omitempty"`
	ResponseSuggestion *string   `json:"response_suggestion,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}
