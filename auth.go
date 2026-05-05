package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
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
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		log.Printf("ERROR: failed to hash password: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	org := Organization{
		ID:           uuid.New(),
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := createOrg(r.Context(), org); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}
		log.Printf("ERROR: failed to create organization: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeSuccess(w, http.StatusCreated, "organization created successfully", org)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	org, err := getOrgByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(org.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	accessToken, err := GenerateAccessToken(org.ID)
	if err != nil {
		log.Printf("ERROR: failed to generate access token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	refreshToken, err := GenerateRefreshToken(org.ID)
	if err != nil {
		log.Printf("ERROR: failed to generate refresh token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeSuccess(w, http.StatusOK, "login successful", map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"org":           org,
	})
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	claims, err := ValidateToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	accessToken, err := GenerateAccessToken(claims.OrgID)
	if err != nil {
		log.Printf("ERROR: failed to generate access token: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeSuccess(w, http.StatusOK, "token refreshed successfully", map[string]any{
		"access_token": accessToken,
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, "logged out successfully", nil)
}
