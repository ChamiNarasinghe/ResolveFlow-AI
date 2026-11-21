package tickets

import (
	"errors"
	"strings"
)

type CreateCommentRequest struct {
	AuthorID   *string `json:"author_id"`
	AuthorType string  `json:"author_type"`
	Body       string  `json:"body"`
	IsInternal bool    `json:"is_internal"`
}

func (r *CreateCommentRequest) Validate() error {
	r.AuthorType = strings.TrimSpace(r.AuthorType)
	r.Body = strings.TrimSpace(r.Body)

	if r.AuthorType == "" {
		return errors.New("author_type is required")
	}

	switch r.AuthorType {
	case "customer", "agent", "system":
	default:
		return errors.New("author_type must be customer, agent, or system")
	}

	if r.Body == "" {
		return errors.New("comment body is required")
	}

	if len(r.Body) > 10000 {
		return errors.New("comment body must not exceed 10000 characters")
	}

	return nil
}