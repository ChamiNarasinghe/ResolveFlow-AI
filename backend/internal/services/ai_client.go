package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ErrInvalidClassification is returned when the AI service response contains
// values outside the known allowed sets.
var ErrInvalidClassification = errors.New("invalid AI classification response")

var (
	validCategories = map[string]bool{"billing": true, "technical_support": true, "account": true, "general_inquiry": true}
	validPriorities = map[string]bool{"low": true, "medium": true, "high": true, "urgent": true}
	validSentiments = map[string]bool{"positive": true, "neutral": true, "negative": true}
)

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

type ClassificationRequest struct {
	Subject     string `json:"subject"`
	Description string `json:"description"`
}

type ClassificationResponse struct {
	Category   string  `json:"category"`
	Priority   string  `json:"priority"`
	Sentiment  string  `json:"sentiment"`
	Confidence float64 `json:"confidence"`
}

// Validate checks that all fields contain known, safe values.
func (c *ClassificationResponse) Validate() error {
	if !validCategories[c.Category] {
		return fmt.Errorf("%w: unknown category %q", ErrInvalidClassification, c.Category)
	}
	if !validPriorities[c.Priority] {
		return fmt.Errorf("%w: unknown priority %q", ErrInvalidClassification, c.Priority)
	}
	if !validSentiments[c.Sentiment] {
		return fmt.Errorf("%w: unknown sentiment %q", ErrInvalidClassification, c.Sentiment)
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("%w: confidence %f out of range [0,1]", ErrInvalidClassification, c.Confidence)
	}
	return nil
}

func NewAIClient(baseURL string) *AIClient {
	return &AIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *AIClient) ClassifyTicket(
	ctx context.Context,
	subject string,
	description string,
) (*ClassificationResponse, error) {

	requestBody := ClassificationRequest{
		Subject:     subject,
		Description: description,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal AI request: %w", err)
	}

	url := c.baseURL + "/api/v1/tickets/classify"

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create AI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"AI service returned status %d",
			resp.StatusCode,
		)
	}

	var result ClassificationResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode AI response: %w", err)
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}

	return &result, nil
}
