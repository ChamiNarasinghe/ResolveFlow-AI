package models

import (
	"encoding/json"
	"time"
)

// TicketHistory represents a single audit record for a ticket change.
type TicketHistory struct {
	ID        string          `json:"id"`
	TicketID  string          `json:"ticket_id"`
	ActorID   *string         `json:"actor_id,omitempty"`
	Action    string          `json:"action"`
	OldValue  json.RawMessage `json:"old_value,omitempty"`
	NewValue  json.RawMessage `json:"new_value,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
