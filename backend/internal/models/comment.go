package models

import "time"

// Comment represents a comment or message attached to a support ticket.
type Comment struct {
    ID         string    `json:"id"`
    TicketID   string    `json:"ticket_id"`
    AuthorID   *string   `json:"author_id,omitempty"`
    AuthorType *string `json:"author_type"`
    Body       string    `json:"body"`
    IsInternal bool      `json:"is_internal"`
    CreatedAt  time.Time `json:"created_at"`
}