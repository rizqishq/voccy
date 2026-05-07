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
	OrgID       uuid.UUID     `json:"org_id"`
	OrgName     string        `json:"-"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	IsActive    bool          `json:"is_active"`
	Settings    BoardSettings `json:"settings"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type PublicBoard struct {
	ID          uuid.UUID     `json:"id"`
	OrgName     string        `json:"org_name"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	IsActive    bool          `json:"is_active"`
	Settings    BoardSettings `json:"settings"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

func toPublicBoard(board Board) PublicBoard {
	return PublicBoard{
		ID:          board.ID,
		OrgName:     board.OrgName,
		Name:        board.Name,
		Slug:        board.Slug,
		Description: board.Description,
		IsActive:    board.IsActive,
		Settings:    board.Settings,
		CreatedAt:   board.CreatedAt,
		UpdatedAt:   board.UpdatedAt,
	}
}

func toPublicBoards(boards []Board) []PublicBoard {
	publicBoards := make([]PublicBoard, 0, len(boards))
	for _, board := range boards {
		publicBoards = append(publicBoards, toPublicBoard(board))
	}
	return publicBoards
}

type Feedback struct {
	ID          uuid.UUID `json:"id"`
	BoardID     uuid.UUID `json:"board_id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	Status      string    `json:"status"`
	VoteCount   int       `json:"vote_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Vote struct {
	ID          uuid.UUID `json:"id"`
	FeedbackID  uuid.UUID `json:"feedback_id"`
	Fingerprint string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

type Organization struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ResponsePayload struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
