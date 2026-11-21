
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
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

	return &result, nil
}
