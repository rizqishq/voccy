package main

import (
	"time"

	"github.com/google/uuid"
)

type BoardSettings struct {
	AllowAnonymous    bool   `json:"allow_anonymous"`
	RequireEmail      bool   `json:"require_email"`
	MaxFeedbackLength int    `json:"max_feedback_length"`
	VotingEnabled     bool   `json:"voting_enabled"`
	Moderation        string `json:"moderation"`
}

type Board struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	IsActive    bool          `json:"is_active"`
	Settings    BoardSettings `json:"settings"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type Feedback struct {
	ID          uuid.UUID `json:"id"`
	BoardID     uuid.UUID `json:"board_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ResponsePayload struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
