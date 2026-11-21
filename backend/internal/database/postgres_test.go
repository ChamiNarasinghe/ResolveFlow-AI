package database

import (
	"context"
	"testing"
)

func TestConnectRejectsEmptyDatabaseURL(t *testing.T) {
	pool, err := Connect(context.Background(), "")

	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("Expected an error for an empty database URL")
	}

	if pool != nil {
		t.Fatal("Expected nil pool for an empty database URL")
	}
}
