package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	HealthHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"Expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response HealthResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected healthy, got %s", response.Status)
	}

	if response.Service != "resolveflow-api" {
		t.Errorf("Unexpected service: %s", response.Service)
	}
}
