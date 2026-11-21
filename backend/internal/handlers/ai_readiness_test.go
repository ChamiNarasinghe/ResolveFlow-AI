package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockAIPinger struct {
	err error
}

func (m mockAIPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestAIReadinessHandlerAvailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health/ai", nil)
	rec := httptest.NewRecorder()

	AIReadinessHandler(mockAIPinger{err: nil})(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAIReadinessHandlerUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health/ai", nil)
	rec := httptest.NewRecorder()

	AIReadinessHandler(mockAIPinger{err: errors.New("connection refused")})(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}
