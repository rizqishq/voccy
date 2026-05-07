package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func voteHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid board slug")
		return
	}

	feedbackID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid feedback id")
		return
	}

	fingerprint := GetFingerprint(r.Context())
	if fingerprint == "" {
		writeError(w, http.StatusBadRequest, "fingerprint required")
		return
	}

	board, err := getBoardBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "board not found")
			return
		}
		log.Printf("ERROR: failed to get board: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get board")
		return
	}

	_, err = getFeedbackByID(r.Context(), feedbackID, board.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "feedback not found")
			return
		}
		log.Printf("ERROR: failed to get feedback: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get feedback")
		return
	}

	voted, err := toggleVote(r.Context(), feedbackID, fingerprint)
	if err != nil {
		log.Printf("ERROR: failed to toggle vote: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to toggle vote")
		return
	}

	if voted {
		writeSuccess(w, http.StatusOK, "vote recorded", nil)
		return
	}

	writeSuccess(w, http.StatusOK, "vote removed", nil)
}
