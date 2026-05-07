package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, "service is healthy", nil)
}

func createBoardHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Settings    *struct {
			AllowAnonymous    *bool   `json:"allow_anonymous"`
			RequireEmail      *bool   `json:"require_email"`
			MaxFeedbackLength *int    `json:"max_feedback_length"`
			VotingEnabled     *bool   `json:"voting_enabled"`
			Moderation        *string `json:"moderation"`
		} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if len(req.Name) > 50 {
		writeError(w, http.StatusBadRequest, "name must be 50 characters or less")
		return
	}

	settings := BoardSettings{
		AllowAnonymous:    true,
		RequireEmail:      false,
		MaxFeedbackLength: 2000,
		VotingEnabled:     true,
		Moderation:        "none",
	}

	if req.Settings != nil {
		if req.Settings.AllowAnonymous != nil {
			settings.AllowAnonymous = *req.Settings.AllowAnonymous
		}
		if req.Settings.RequireEmail != nil {
			settings.RequireEmail = *req.Settings.RequireEmail
		}
		if req.Settings.MaxFeedbackLength != nil {
			settings.MaxFeedbackLength = *req.Settings.MaxFeedbackLength
		}
		if req.Settings.VotingEnabled != nil {
			settings.VotingEnabled = *req.Settings.VotingEnabled
		}
		if req.Settings.Moderation != nil {
			settings.Moderation = *req.Settings.Moderation
		}
	}

	board := Board{
		ID:          uuid.New(),
		OrgID:       GetOrgIDFromContext(r.Context()),
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		Description: req.Description,
		IsActive:    true,
		Settings:    settings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := createBoard(r.Context(), board); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "board with a similar name already exists")
			return
		}
		log.Printf("ERROR: failed to create board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create board")
		return
	}

	writeSuccess(w, http.StatusCreated, "board created successfully", board)
}

func listBoardsHandler(w http.ResponseWriter, r *http.Request) {
	boards, err := getBoards(r.Context())
	if err != nil {
		log.Printf("ERROR: failed to fetch boards: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch boards")
		return
	}

	writeSuccess(w, http.StatusOK, "boards retrieved successfully", toPublicBoards(boards))
}

func getBoardByIDHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	board, err := getBoardByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		log.Printf("ERROR: failed to get board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get board")
		return
	}

	writeSuccess(w, http.StatusOK, "board retrieved successfully", toPublicBoard(*board))
}

func updateBoardHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Settings    *struct {
			AllowAnonymous    *bool   `json:"allow_anonymous"`
			RequireEmail      *bool   `json:"require_email"`
			MaxFeedbackLength *int    `json:"max_feedback_length"`
			VotingEnabled     *bool   `json:"voting_enabled"`
			Moderation        *string `json:"moderation"`
		} `json:"settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if len(req.Name) > 50 {
		writeError(w, http.StatusBadRequest, "name must be 50 characters or less")
		return
	}

	orgID := GetOrgIDFromContext(r.Context())
	existing, err := getBoardByIDForOrg(r.Context(), id, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		log.Printf("ERROR: failed to get board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get board")
		return
	}

	settings := existing.Settings
	if req.Settings != nil {
		if req.Settings.AllowAnonymous != nil {
			settings.AllowAnonymous = *req.Settings.AllowAnonymous
		}
		if req.Settings.RequireEmail != nil {
			settings.RequireEmail = *req.Settings.RequireEmail
		}
		if req.Settings.MaxFeedbackLength != nil {
			settings.MaxFeedbackLength = *req.Settings.MaxFeedbackLength
		}
		if req.Settings.VotingEnabled != nil {
			settings.VotingEnabled = *req.Settings.VotingEnabled
		}
		if req.Settings.Moderation != nil {
			settings.Moderation = *req.Settings.Moderation
		}
	}

	board, err := updateBoardForOrg(r.Context(), id, orgID, req.Name, req.Description, settings)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "board with a similar name already exists")
			return
		}
		log.Printf("ERROR: failed to update board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update board")
		return
	}

	writeSuccess(w, http.StatusOK, "board updated successfully", board)
}

func deleteBoardHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	err = deleteBoardForOrg(r.Context(), id, GetOrgIDFromContext(r.Context()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		log.Printf("ERROR: failed to delete board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete board")
		return
	}

	writeSuccess(w, http.StatusOK, "board deleted successfully", nil)
}

func createFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	_, err = getBoardByID(r.Context(), boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		log.Printf("ERROR: failed to verify board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to verify board")
		return
	}

	var req struct {
		Title       string `json:"title"`
		Body        string `json:"body"`
		AuthorName  string `json:"author_name"`
		AuthorEmail string `json:"author_email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	feedback := Feedback{
		ID:          uuid.New(),
		BoardID:     boardID,
		Title:       req.Title,
		Body:        req.Body,
		AuthorName:  req.AuthorName,
		AuthorEmail: req.AuthorEmail,
		Status:      "open",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := createFeedback(r.Context(), feedback); err != nil {
		log.Printf("ERROR: failed to create feedback: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create feedback")
		return
	}

	writeSuccess(w, http.StatusCreated, "feedback created successfully", feedback)
}

func listFeedbacksHandler(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	_, err = getBoardByID(r.Context(), boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		log.Printf("ERROR: failed to verify board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to verify board")
		return
	}

	feedbacks, err := getFeedbacksByBoardID(r.Context(), boardID)
	if err != nil {
		log.Printf("ERROR: failed to fetch feedbacks: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to fetch feedbacks")
		return
	}

	writeSuccess(w, http.StatusOK, "feedbacks retrieved successfully", feedbacks)
}

func getFeedbackByIDHandler(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "feedbackId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid feedback id")
		return
	}

	feedback, err := getFeedbackByID(r.Context(), id, boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "feedback not found")
			return
		}
		log.Printf("ERROR: failed to get feedback: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get feedback")
		return
	}

	writeSuccess(w, http.StatusOK, "feedback retrieved successfully", feedback)
}

func updateFeedbackStatusHandler(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "feedbackId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid feedback id")
		return
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	validStatuses := map[string]bool{"open": true, "in_progress": true, "resolved": true, "closed": true}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid status, must be: open, in_progress, resolved, or closed")
		return
	}

	feedback, err := updateFeedbackStatusForOrg(r.Context(), id, boardID, GetOrgIDFromContext(r.Context()), req.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "feedback not found")
			return
		}
		log.Printf("ERROR: failed to update feedback: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update feedback")
		return
	}

	writeSuccess(w, http.StatusOK, "feedback updated successfully", feedback)
}

func deleteFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	boardID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid board id")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "feedbackId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid feedback id")
		return
	}

	err = deleteFeedbackForOrg(r.Context(), id, boardID, GetOrgIDFromContext(r.Context()))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "feedback not found")
			return
		}
		log.Printf("ERROR: failed to delete feedback: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete feedback")
		return
	}

	writeSuccess(w, http.StatusOK, "feedback deleted successfully", nil)
}

func getOrgHandler(w http.ResponseWriter, r *http.Request) {
	orgID := GetOrgIDFromContext(r.Context())
	org, err := getOrgByID(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "organization not found")
			return
		}
		log.Printf("ERROR: failed to get organization: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get organization")
		return
	}

	writeSuccess(w, http.StatusOK, "organization retrieved successfully", org)
}

func updateOrgHandler(w http.ResponseWriter, r *http.Request) {
	orgID := GetOrgIDFromContext(r.Context())

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}

	org, err := updateOrg(r.Context(), orgID, req.Name, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "organization not found")
			return
		}
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "email already in use")
			return
		}
		log.Printf("ERROR: failed to update organization: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update organization")
		return
	}

	writeSuccess(w, http.StatusOK, "organization updated successfully", org)
}
