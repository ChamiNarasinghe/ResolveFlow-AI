package tickets

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrEmptyUpdate = errors.New("at least one field must be provided")

// OptionalString tracks whether a field was supplied,
// including when its JSON value is null.
type OptionalString struct {
	Set   bool
	Value *string
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	o.Value = &value
	return nil
}

type UpdateTicketRequest struct {
	Status         OptionalString `json:"status"`
	Priority       OptionalString `json:"priority"`
	Category       OptionalString `json:"category"`
	AssignedTeamID OptionalString `json:"assigned_team_id"`
	AssignedTo     OptionalString `json:"assigned_to"`
}

func (req *UpdateTicketRequest) Validate() error {
	if !req.Status.Set &&
		!req.Priority.Set &&
		!req.Category.Set &&
		!req.AssignedTeamID.Set &&
		!req.AssignedTo.Set {
		return ErrEmptyUpdate
	}

	if req.Status.Set {
		if req.Status.Value == nil {
			return errors.New("status cannot be null")
		}

		value := strings.TrimSpace(*req.Status.Value)

		switch value {
		case "open", "in_progress", "waiting_for_customer", "resolved", "closed":
			req.Status.Value = &value
		default:
			return errors.New("invalid ticket status")
		}
	}

	if req.Priority.Set {
		if req.Priority.Value == nil {
			return errors.New("priority cannot be null")
		}

		value := strings.TrimSpace(*req.Priority.Value)

		switch value {
		case "low", "medium", "high", "urgent":
			req.Priority.Value = &value
		default:
			return errors.New("invalid ticket priority")
		}
	}

	if req.Category.Set && req.Category.Value != nil {
		value := strings.TrimSpace(*req.Category.Value)

		if len(value) > 100 {
			return errors.New("category must be at most 100 characters")
		}

		req.Category.Value = &value
	}

	if err := validateOptionalUUID(&req.AssignedTeamID); err != nil {
		return errors.New("invalid assigned_team_id")
	}

	if err := validateOptionalUUID(&req.AssignedTo); err != nil {
		return errors.New("invalid assigned_to")
	}

	return nil
}

func validateOptionalUUID(field *OptionalString) error {
	if !field.Set || field.Value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*field.Value)

	if trimmed == "" {
		return errors.New("UUID cannot be empty")
	}

	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return err
	}

	value := parsed.String()
	field.Value = &value

	return nil
}
