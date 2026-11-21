package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockDatabase struct {
	err error
}

func (m mockDatabase) Ping(ctx context.Context) error {
	return m.err
}

func TestReadinessHandlerDatabaseConnected(t *testing.T) {
	db := mockDatabase{err: nil}

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	recorder := httptest.NewRecorder()

	ReadinessHandler(db)(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

func TestReadinessHandlerDatabaseUnavailable(t *testing.T) {
	db := mockDatabase{err: errors.New("database unavailable")}

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	recorder := httptest.NewRecorder()

	ReadinessHandler(db)(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", recorder.Code)
	}
}
