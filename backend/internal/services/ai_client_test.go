package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIClientClassifyTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}

			if r.URL.Path != "/api/v1/tickets/classify" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			var request ClassificationRequest

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			if request.Subject != "Payment failure" {
				t.Errorf("unexpected subject: %s", request.Subject)
			}

			if request.Description != "My payment failed twice." {
				t.Errorf("unexpected description: %s", request.Description)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			_, _ = w.Write([]byte(`{
				"category": "billing",
				"priority": "high",
				"sentiment": "negative",
				"confidence": 0.8
			}`))
		},
	))

	defer server.Close()

	client := NewAIClient(server.URL, "")

	result, err := client.ClassifyTicket(
		context.Background(),
		"Payment failure",
		"My payment failed twice.",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Category != "billing" {
		t.Errorf("expected billing, got %s", result.Category)
	}

	if result.Priority != "high" {
		t.Errorf("expected high, got %s", result.Priority)
	}

	if result.Sentiment != "negative" {
		t.Errorf("expected negative, got %s", result.Sentiment)
	}

	if result.Confidence != 0.8 {
		t.Errorf("expected confidence 0.8, got %f", result.Confidence)
	}
}

func TestAIClientHandlesServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "AI service unavailable", http.StatusInternalServerError)
		},
	))

	defer server.Close()

	client := NewAIClient(server.URL, "")

	_, err := client.ClassifyTicket(
		context.Background(),
		"Payment failure",
		"My payment failed.",
	)

	if err == nil {
		t.Fatal("expected an error when AI service returns HTTP 500")
	}
}

func TestAIClientRejectsUnknownCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"category":"unknown_cat","priority":"high","sentiment":"negative","confidence":0.9}`))
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "").ClassifyTicket(context.Background(), "s", "d")
	if !errors.Is(err, ErrInvalidClassification) {
		t.Errorf("expected ErrInvalidClassification, got %v", err)
	}
}

func TestAIClientRejectsUnknownPriority(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"category":"billing","priority":"critical","sentiment":"negative","confidence":0.9}`))
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "").ClassifyTicket(context.Background(), "s", "d")
	if !errors.Is(err, ErrInvalidClassification) {
		t.Errorf("expected ErrInvalidClassification, got %v", err)
	}
}

func TestAIClientRejectsUnknownSentiment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"category":"billing","priority":"high","sentiment":"angry","confidence":0.9}`))
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "").ClassifyTicket(context.Background(), "s", "d")
	if !errors.Is(err, ErrInvalidClassification) {
		t.Errorf("expected ErrInvalidClassification, got %v", err)
	}
}

func TestAIClientRejectsOutOfRangeConfidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"category":"billing","priority":"high","sentiment":"negative","confidence":1.5}`))
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "").ClassifyTicket(context.Background(), "s", "d")
	if !errors.Is(err, ErrInvalidClassification) {
		t.Errorf("expected ErrInvalidClassification, got %v", err)
	}
}

func TestAIClientSendsAPIKeyHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "secret-key" {
			t.Errorf("expected X-API-Key: secret-key, got %q", r.Header.Get("X-API-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"category":"billing","priority":"high","sentiment":"negative","confidence":0.9}`))
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "secret-key").ClassifyTicket(context.Background(), "s", "d")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAIClientOmitsAPIKeyHeaderWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "" {
			t.Errorf("expected no X-API-Key header, got %q", r.Header.Get("X-API-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"category":"billing","priority":"high","sentiment":"negative","confidence":0.9}`))
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "").ClassifyTicket(context.Background(), "s", "d")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAIClientReturnsErrAIUnauthorizedOn401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "bad-key").ClassifyTicket(context.Background(), "s", "d")
	if !errors.Is(err, ErrAIUnauthorized) {
		t.Errorf("expected ErrAIUnauthorized, got %v", err)
	}
}

func TestAIClientReturnsErrAIUnauthorizedOn403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := NewAIClient(server.URL, "bad-key").ClassifyTicket(context.Background(), "s", "d")
	if !errors.Is(err, ErrAIUnauthorized) {
		t.Errorf("expected ErrAIUnauthorized, got %v", err)
	}
}

func TestClassificationResponseValidateAcceptsValidResponse(t *testing.T) {
	r := &ClassificationResponse{
		Category:   "technical_support",
		Priority:   "urgent",
		Sentiment:  "neutral",
		Confidence: 0.75,
	}
	if err := r.Validate(); err != nil {
		t.Errorf("expected no error for valid response, got %v", err)
	}
}
