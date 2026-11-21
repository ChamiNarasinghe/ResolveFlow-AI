package tickets

import (
	"strings"
	"testing"
)

func optionalString(value string) OptionalString {
	return OptionalString{
		Set:   true,
		Value: &value,
	}
}

func TestUpdateTicketRequestValid(t *testing.T) {
	req := UpdateTicketRequest{
		Status:   optionalString("in_progress"),
		Priority: optionalString("high"),
		Category: optionalString("Payment"),
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}
}

func TestUpdateTicketRequestEmpty(t *testing.T) {
	req := UpdateTicketRequest{}

	if err := req.Validate(); err != ErrEmptyUpdate {
		t.Errorf("expected ErrEmptyUpdate, got %v", err)
	}
}

func TestUpdateTicketRequestInvalidStatus(t *testing.T) {
	req := UpdateTicketRequest{
		Status: optionalString("pending"),
	}

	if err := req.Validate(); err == nil {
		t.Error("expected invalid status error")
	}
}

func TestUpdateTicketRequestInvalidPriority(t *testing.T) {
	req := UpdateTicketRequest{
		Priority: optionalString("critical"),
	}

	if err := req.Validate(); err == nil {
		t.Error("expected invalid priority error")
	}
}

func TestUpdateTicketRequestCategoryTooLong(t *testing.T) {
	req := UpdateTicketRequest{
		Category: optionalString(strings.Repeat("a", 101)),
	}

	if err := req.Validate(); err == nil {
		t.Error("expected category length error")
	}
}

func TestUpdateTicketRequestInvalidAssignedTeamID(t *testing.T) {
	req := UpdateTicketRequest{
		AssignedTeamID: optionalString("not-a-uuid"),
	}

	if err := req.Validate(); err == nil {
		t.Error("expected invalid assigned_team_id error")
	}
}

func TestUpdateTicketRequestInvalidAssignedTo(t *testing.T) {
	req := UpdateTicketRequest{
		AssignedTo: optionalString("not-a-uuid"),
	}

	if err := req.Validate(); err == nil {
		t.Error("expected invalid assigned_to error")
	}
}

func TestUpdateTicketRequestTrimsValues(t *testing.T) {
	req := UpdateTicketRequest{
		Status:   optionalString(" in_progress "),
		Priority: optionalString(" high "),
		Category: optionalString(" Payment "),
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got error: %v", err)
	}

	if *req.Status.Value != "in_progress" {
		t.Errorf("expected trimmed status, got %q", *req.Status.Value)
	}

	if *req.Priority.Value != "high" {
		t.Errorf("expected trimmed priority, got %q", *req.Priority.Value)
	}

	if *req.Category.Value != "Payment" {
		t.Errorf("expected trimmed category, got %q", *req.Category.Value)
	}
}

func TestUpdateTicketRequestNullCategory(t *testing.T) {
	req := UpdateTicketRequest{
		Category: OptionalString{
			Set:   true,
			Value: nil,
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected null category to be valid, got %v", err)
	}
}

func TestUpdateTicketRequestNullStatus(t *testing.T) {
	req := UpdateTicketRequest{
		Status: OptionalString{
			Set:   true,
			Value: nil,
		},
	}

	if err := req.Validate(); err == nil {
		t.Error("expected null status to be rejected")
	}
}
