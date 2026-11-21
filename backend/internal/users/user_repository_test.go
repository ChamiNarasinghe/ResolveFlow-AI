package users

import (
	"errors"
	"testing"
)

// TestErrEmailTakenIsSentinel verifies that ErrEmailTaken is a stable sentinel
// that can be matched with errors.Is across package boundaries.
func TestErrEmailTakenIsSentinel(t *testing.T) {
	wrapped := errors.Join(ErrEmailTaken, errors.New("extra context"))

	if !errors.Is(wrapped, ErrEmailTaken) {
		t.Error("errors.Is must match ErrEmailTaken through wrapping")
	}
}
