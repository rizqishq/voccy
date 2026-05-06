package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/crypto/bcrypt"
)

func setupTestDB(t *testing.T) {
	t.Helper()

	dbURL := os.Getenv("TEST_DB_URL")
	if dbURL == "" {
		t.Skip("TEST_DB_URL not set, skipping integration test")
	}

	var err error
	pool, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("failed to ping test db: %v", err)
	}

	os.Setenv("JWT_SECRET", "test-secret")
}

func cleanupDB(t *testing.T) {
	t.Helper()
	pool.Exec(context.Background(), "DELETE FROM feedbacks")
	pool.Exec(context.Background(), "DELETE FROM boards")
	pool.Exec(context.Background(), "DELETE FROM organizations")
}

func newTestRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/health", healthHandler)

	r.Post("/auth/register", registerHandler)
	r.Post("/auth/login", loginHandler)
	r.Post("/auth/refresh", refreshHandler)
	r.Delete("/auth/logout", logoutHandler)

	r.Get("/boards", listBoardsHandler)
	r.Get("/boards/{id}", getBoardByIDHandler)
	r.Get("/boards/{id}/feedbacks", listFeedbacksHandler)
	r.Get("/boards/{id}/feedbacks/{feedbackId}", getFeedbackByIDHandler)

	r.Group(func(r chi.Router) {
		r.Use(AuthMiddleware)

		r.Post("/boards", createBoardHandler)
		r.Put("/boards/{id}", updateBoardHandler)
		r.Delete("/boards/{id}", deleteBoardHandler)

		r.Post("/boards/{id}/feedbacks", createFeedbackHandler)
		r.Patch("/boards/{id}/feedbacks/{feedbackId}", updateFeedbackStatusHandler)
		r.Delete("/boards/{id}/feedbacks/{feedbackId}", deleteFeedbackHandler)
	})

	return r
}

func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) ResponsePayload {
	t.Helper()
	var resp ResponsePayload
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp
}

func createTestOrg(t *testing.T) (Organization, string) {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	now := time.Now()
	org := Organization{
		ID:           uuid.New(),
		Name:         "Test Org",
		Email:        uuid.New().String()[:8] + "@example.com",
		PasswordHash: string(passwordHash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := createOrg(context.Background(), org); err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	token, err := GenerateAccessToken(org.ID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return org, token
}

func authRequest(req *http.Request, token string) *http.Request {
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func TestHealthHandler_Success(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}
	if resp.Message != "service is healthy" {
		t.Errorf("expected message 'service is healthy', got '%s'", resp.Message)
	}
}

func TestCreateBoard_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	body := `{"name": "Test Board", "description": "A test board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}
	if resp.Data == nil {
		t.Fatal("expected data to be non-nil")
	}
}
func TestCreateBoard_WithSettings(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	body := `{
		"name": "Board With Settings",
		"description": "Custom settings",
		"settings": {
			"allow_anonymous": false,
			"require_email": true,
			"max_feedback_length": 500,
			"voting_enabled": false,
			"moderation": "pre"
		}
	}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	settings, ok := data["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("expected settings to be a map")
	}

	if settings["allow_anonymous"] != false {
		t.Errorf("expected allow_anonymous=false, got %v", settings["allow_anonymous"])
	}
	if settings["require_email"] != true {
		t.Errorf("expected require_email=true, got %v", settings["require_email"])
	}
	if settings["max_feedback_length"] != float64(500) {
		t.Errorf("expected max_feedback_length=500, got %v", settings["max_feedback_length"])
	}
	if settings["voting_enabled"] != false {
		t.Errorf("expected voting_enabled=false, got %v", settings["voting_enabled"])
	}
	if settings["moderation"] != "pre" {
		t.Errorf("expected moderation='pre', got %v", settings["moderation"])
	}
}
func TestCreateBoard_InvalidJSON(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", resp.Status)
	}
	if resp.Message != "invalid json" {
		t.Errorf("expected message 'invalid json', got '%s'", resp.Message)
	}
}
func TestCreateBoard_MissingName(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	body := `{"description": "no name provided"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "name is required" {
		t.Errorf("expected message 'name is required', got '%s'", resp.Message)
	}
}
func TestCreateBoard_NameTooLong(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	longName := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 51 chars
	body := `{"name": "` + longName + `"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "name must be 50 characters or less" {
		t.Errorf("expected message 'name must be 50 characters or less', got '%s'", resp.Message)
	}
}
func TestCreateBoard_DuplicateName(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	body := `{"name": "Duplicate Board"}`

	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("first create failed with status %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "board with a similar name already exists" {
		t.Errorf("expected conflict message, got '%s'", resp.Message)
	}
}
func TestListBoards_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "List Test Board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/boards", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected data to be an array")
	}
	if len(data) == 0 {
		t.Error("expected at least one board in list")
	}
}
func TestListBoards_Empty(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/boards", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected data to be an array")
	}
	if len(data) != 0 {
		t.Errorf("expected empty array, got %d items", len(data))
	}
}

func TestGetBoardByID_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "Get By ID Board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	boardID := data["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/boards/"+boardID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp = parseResponse(t, rec)
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}

	boardData := resp.Data.(map[string]interface{})
	if boardData["name"] != "Get By ID Board" {
		t.Errorf("expected name 'Get By ID Board', got '%s'", boardData["name"])
	}
	if boardData["slug"] != "get-by-id-board" {
		t.Errorf("expected slug 'get-by-id-board', got '%s'", boardData["slug"])
	}
}
func TestGetBoardByID_InvalidID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/boards/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "invalid board id" {
		t.Errorf("expected message 'invalid board id', got '%s'", resp.Message)
	}
}

func TestGetBoardByID_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	r := newTestRouter()

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/boards/"+fakeID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "board not found" {
		t.Errorf("expected message 'board not found', got '%s'", resp.Message)
	}
}

func TestUpdateBoard_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "Original Name", "description": "original"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	boardID := data["id"].(string)

	updateBody := `{"name": "Updated Name", "description": "updated"}`
	req = httptest.NewRequest(http.MethodPut, "/boards/"+boardID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp = parseResponse(t, rec)
	boardData := resp.Data.(map[string]interface{})
	if boardData["name"] != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", boardData["name"])
	}
	if boardData["slug"] != "updated-name" {
		t.Errorf("expected slug 'updated-name', got '%s'", boardData["slug"])
	}
	if boardData["description"] != "updated" {
		t.Errorf("expected description 'updated', got '%s'", boardData["description"])
	}
}
func TestUpdateBoard_WithSettings(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "Settings Board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	boardID := data["id"].(string)

	updateBody := `{"name": "Settings Board", "description": "", "settings": {"allow_anonymous": false}}`
	req = httptest.NewRequest(http.MethodPut, "/boards/"+boardID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp = parseResponse(t, rec)
	boardData := resp.Data.(map[string]interface{})
	settings := boardData["settings"].(map[string]interface{})

	if settings["allow_anonymous"] != false {
		t.Errorf("expected allow_anonymous=false, got %v", settings["allow_anonymous"])
	}
	if settings["voting_enabled"] != true {
		t.Errorf("expected voting_enabled=true (unchanged), got %v", settings["voting_enabled"])
	}
}
func TestUpdateBoard_InvalidID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "Whatever"}`
	req := httptest.NewRequest(http.MethodPut, "/boards/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestUpdateBoard_InvalidJSON(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPut, "/boards/"+fakeID, bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestUpdateBoard_MissingName(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	fakeID := uuid.New().String()
	body := `{"description": "no name"}`
	req := httptest.NewRequest(http.MethodPut, "/boards/"+fakeID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "name is required" {
		t.Errorf("expected 'name is required', got '%s'", resp.Message)
	}
}
func TestUpdateBoard_NameTooLong(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	fakeID := uuid.New().String()
	longName := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" // 51 chars
	body := `{"name": "` + longName + `"}`
	req := httptest.NewRequest(http.MethodPut, "/boards/"+fakeID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestUpdateBoard_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	fakeID := uuid.New().String()
	body := `{"name": "Nonexistent"}`
	req := httptest.NewRequest(http.MethodPut, "/boards/"+fakeID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}
func TestDeleteBoard_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "Delete Me Board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	boardID := data["id"].(string)

	req = httptest.NewRequest(http.MethodDelete, "/boards/"+boardID, nil)
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/boards/"+boardID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", rec.Code)
	}
}
func TestDeleteBoard_InvalidID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/boards/not-a-uuid", nil)
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestDeleteBoard_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/boards/"+fakeID, nil)
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func createTestBoard(t *testing.T, r *chi.Mux, token string) string {
	t.Helper()
	body := `{"name": "Feedback Test Board ` + uuid.New().String()[:8] + `"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create test board: status %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	return data["id"].(string)
}
func TestCreateFeedback_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Bug Report", "body": "Something is broken", "author_name": "John", "author_email": "john@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}

	data := resp.Data.(map[string]interface{})
	if data["title"] != "Bug Report" {
		t.Errorf("expected title 'Bug Report', got '%s'", data["title"])
	}
	if data["status"] != "open" {
		t.Errorf("expected status 'open', got '%s'", data["status"])
	}
	if data["board_id"] != boardID {
		t.Errorf("expected board_id '%s', got '%s'", boardID, data["board_id"])
	}
}
func TestCreateFeedback_InvalidBoardID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"title": "Test", "body": "Test body"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/not-a-uuid/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestCreateFeedback_BoardNotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	fakeID := uuid.New().String()
	body := `{"title": "Test", "body": "Test body"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+fakeID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}
func TestCreateFeedback_InvalidJSON(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestCreateFeedback_MissingTitle(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"body": "No title here"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "title is required" {
		t.Errorf("expected 'title is required', got '%s'", resp.Message)
	}
}
func TestCreateFeedback_MissingBody(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Has title but no body"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	if resp.Message != "body is required" {
		t.Errorf("expected 'body is required', got '%s'", resp.Message)
	}
}
func TestListFeedbacks_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Feedback 1", "body": "Body 1"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected data to be an array")
	}
	if len(data) != 1 {
		t.Errorf("expected 1 feedback, got %d", len(data))
	}
}
func TestListFeedbacks_InvalidBoardID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/boards/not-a-uuid/feedbacks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestListFeedbacks_BoardNotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	r := newTestRouter()

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/boards/"+fakeID+"/feedbacks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestListFeedbacks_Empty(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected data to be an array")
	}
	if len(data) != 0 {
		t.Errorf("expected 0 feedbacks, got %d", len(data))
	}
}
func TestGetFeedbackByID_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Get Me", "body": "Find this feedback"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	feedbackID := data["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks/"+feedbackID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp = parseResponse(t, rec)
	feedbackData := resp.Data.(map[string]interface{})
	if feedbackData["title"] != "Get Me" {
		t.Errorf("expected title 'Get Me', got '%s'", feedbackData["title"])
	}
}
func TestGetFeedbackByID_InvalidID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestGetFeedbackByID_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks/"+fakeID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}
func TestUpdateFeedbackStatus_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Status Test", "body": "Will change status"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	feedbackID := data["id"].(string)

	updateBody := `{"status": "in_progress"}`
	req = httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/feedbacks/"+feedbackID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	resp = parseResponse(t, rec)
	feedbackData := resp.Data.(map[string]interface{})
	if feedbackData["status"] != "in_progress" {
		t.Errorf("expected status 'in_progress', got '%s'", feedbackData["status"])
	}
}
func TestUpdateFeedbackStatus_AllValidStatuses(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	statuses := []string{"open", "in_progress", "resolved", "closed"}

	for _, status := range statuses {
		t.Run("status_"+status, func(t *testing.T) {
			body := `{"title": "Status ` + status + `", "body": "body"}`
			req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			req = authRequest(req, token)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			resp := parseResponse(t, rec)
			data := resp.Data.(map[string]interface{})
			feedbackID := data["id"].(string)

			updateBody := `{"status": "` + status + `"}`
			req = httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/feedbacks/"+feedbackID, bytes.NewBufferString(updateBody))
			req.Header.Set("Content-Type", "application/json")
			req = authRequest(req, token)
			rec = httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200 for '%s', got %d", status, rec.Code)
			}

			resp = parseResponse(t, rec)
			feedbackData := resp.Data.(map[string]interface{})
			if feedbackData["status"] != status {
				t.Errorf("expected status '%s', got '%s'", status, feedbackData["status"])
			}
		})
	}
}
func TestUpdateFeedbackStatus_InvalidStatus(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Invalid Status", "body": "body"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	feedbackID := data["id"].(string)

	updateBody := `{"status": "invalid_status"}`
	req = httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/feedbacks/"+feedbackID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	resp = parseResponse(t, rec)
	if resp.Message != "invalid status, must be: open, in_progress, resolved, or closed" {
		t.Errorf("unexpected message: '%s'", resp.Message)
	}
}
func TestUpdateFeedbackStatus_InvalidFeedbackID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"status": "closed"}`
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/feedbacks/not-a-uuid", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestUpdateFeedbackStatus_InvalidJSON(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/feedbacks/"+fakeID, bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestUpdateFeedbackStatus_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	fakeID := uuid.New().String()
	body := `{"status": "closed"}`
	req := httptest.NewRequest(http.MethodPatch, "/boards/"+boardID+"/feedbacks/"+fakeID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}
func TestDeleteFeedback_Success(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Delete Me", "body": "Will be deleted"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	feedbackID := data["id"].(string)

	req = httptest.NewRequest(http.MethodDelete, "/boards/"+boardID+"/feedbacks/"+feedbackID, nil)
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks/"+feedbackID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404 after delete, got %d", rec.Code)
	}
}
func TestDeleteFeedback_InvalidID(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	req := httptest.NewRequest(http.MethodDelete, "/boards/"+boardID+"/feedbacks/not-a-uuid", nil)
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestDeleteFeedback_NotFound(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	fakeID := uuid.New().String()
	req := httptest.NewRequest(http.MethodDelete, "/boards/"+boardID+"/feedbacks/"+fakeID, nil)
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}
func TestDeleteBoard_CascadeDeletesFeedbacks(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Cascade Test", "body": "Should be deleted with board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	feedbackID := data["id"].(string)

	req = httptest.NewRequest(http.MethodDelete, "/boards/"+boardID, nil)
	req = authRequest(req, token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/boards/"+boardID+"/feedbacks/"+feedbackID, nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected feedback to be deleted (404), got %d", rec.Code)
	}
}
func TestCreateBoard_EmptyBody(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
func TestCreateFeedback_WithoutAuthor(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()
	boardID := createTestBoard(t, r, token)

	body := `{"title": "Anonymous Feedback", "body": "No author info"}`
	req := httptest.NewRequest(http.MethodPost, "/boards/"+boardID+"/feedbacks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	if data["author_name"] != "" {
		t.Errorf("expected empty author_name, got '%s'", data["author_name"])
	}
}
func TestCreateBoard_SpecialCharactersInName(t *testing.T) {
	setupTestDB(t)
	defer cleanupDB(t)

	_, token := createTestOrg(t)
	r := newTestRouter()

	body := `{"name": "Hello World! @#$% Test"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = authRequest(req, token)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rec.Code)
	}

	resp := parseResponse(t, rec)
	data := resp.Data.(map[string]interface{})
	if data["slug"] != "hello-world-test" {
		t.Errorf("expected slug 'hello-world-test', got '%s'", data["slug"])
	}
}
func TestProtectedEndpoint_NoToken(t *testing.T) {
	r := newTestRouter()
	body := `{"name": "Unauthorized Board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestProtectedEndpoint_InvalidToken(t *testing.T) {
	r := newTestRouter()
	body := `{"name": "Unauthorized Board"}`
	req := httptest.NewRequest(http.MethodPost, "/boards", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestContentTypeJSON(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}
