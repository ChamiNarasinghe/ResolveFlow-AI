
package tickets

import (
	"errors"
	"net/mail"
	"strings"
)

var (
	ErrInvalidSubject      = errors.New("subject is required and must be at most 255 characters")
	ErrInvalidDescription  = errors.New("description is required")
	ErrInvalidCustomerName = errors.New("customer name is required")
	ErrInvalidCustomerEmail = errors.New("valid customer email is required")
)

// CreateTicketRequest represents the data required to create a ticket.
type CreateTicketRequest struct {
	Subject       string `json:"subject"`
	Description   string `json:"description"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	Channel       string `json:"channel"`
}

// Validate checks whether the ticket creation request is valid.
func (req *CreateTicketRequest) Validate() error {
	req.Subject = strings.TrimSpace(req.Subject)
  req.Description = strings.TrimSpace(req.Description)
  req.CustomerName = strings.TrimSpace(req.CustomerName)
  req.CustomerEmail = strings.TrimSpace(req.CustomerEmail)
  req.Channel = strings.TrimSpace(req.Channel)

	if req.Subject == "" || len(req.Subject) > 255 {
		return ErrInvalidSubject
	}

	if req.Description == "" {
		return ErrInvalidDescription
	}

	if req.CustomerName == "" {
		return ErrInvalidCustomerName
	}

	if _, err := mail.ParseAddress(req.CustomerEmail); err != nil {
		return ErrInvalidCustomerEmail
	}

	if req.Channel == "" {
		req.Channel = "web"
	}

	switch req.Channel {
	case "web", "email", "chat", "phone":
		return nil
	default:
		return errors.New("invalid ticket channel")
	}
}
