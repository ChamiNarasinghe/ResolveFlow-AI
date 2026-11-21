package models

import "time"

// AIPrediction represents an AI-generated ticket classification.
type AIPrediction struct {
	ID                string    `json:"id"`
	TicketID          string    `json:"ticket_id"`
	ModelVersion      string    `json:"model_version"`
	Category          string    `json:"category"`
	PredictedPriority string    `json:"predicted_priority"`
	Sentiment         string    `json:"sentiment"`
	Confidence        float64   `json:"confidence"`
	CreatedAt         time.Time `json:"created_at"`
}
