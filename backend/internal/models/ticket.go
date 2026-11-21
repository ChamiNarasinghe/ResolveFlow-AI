
package models

import "time"

// Ticket represents a customer support ticket.
type Ticket struct {
    ID             string     `json:"id"`
    TicketNumber   int64      `json:"ticket_number"`
    Subject        string     `json:"subject"`
    Description    string     `json:"description"`
    CustomerName   string     `json:"customer_name"`
    CustomerEmail  string     `json:"customer_email"`
    Status         string     `json:"status"`
    Priority       string     `json:"priority"`
    Channel        string     `json:"channel"`
    Category       *string    `json:"category,omitempty"`
    AssignedTeamID *string    `json:"assigned_team_id,omitempty"`
    AssignedTo     *string    `json:"assigned_to,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at"`
    ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}
