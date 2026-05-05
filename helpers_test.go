package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestGenerateSlug_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "Hello World", "hello-world"},
		{"single word", "Feedback", "feedback"},
		{"already lowercase", "my board", "my-board"},
		{"multiple spaces", "My   Board", "my-board"},
		{"special characters", "Hello! @World# $Test%", "hello-world-test"},
		{"leading trailing spaces", "  trimmed  ", "trimmed"},
		{"numbers", "Board 123", "board-123"},
		{"dashes preserved", "my-board-name", "my-board-name"},
		{"mixed special and spaces", "Test & Board!", "test-board"},
		{"unicode stripped", "Café Board", "caf-board"},
		{"all special chars", "!@#$%^&*()", ""},
		{"leading trailing dashes", "---hello---", "hello"},
		{"multiple dashes collapsed", "hello---world", "hello-world"},
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWriteSuccess_WithData(t *testing.T) {
	rec := httptest.NewRecorder()

	data := map[string]string{"key": "value"}
	writeSuccess(rec, 200, "ok", data)

	if rec.Code != 200 {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp ResponsePayload
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", resp.Status)
	}
	if resp.Message != "ok" {
		t.Errorf("expected message 'ok', got '%s'", resp.Message)
	}
	if resp.Data == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestWriteSuccess_WithNilData(t *testing.T) {
	rec := httptest.NewRecorder()

	writeSuccess(rec, 200, "done", nil)

	var resp ResponsePayload
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Data != nil {
		t.Errorf("expected data to be nil (omitempty), got %v", resp.Data)
	}
}

func TestWriteSuccess_StatusCodes(t *testing.T) {
	codes := []int{200, 201, 204}

	for _, code := range codes {
		t.Run("status_"+string(rune(code)), func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeSuccess(rec, code, "msg", nil)

			if rec.Code != code {
				t.Errorf("expected status %d, got %d", code, rec.Code)
			}
		})
	}
}

func TestWriteError_Basic(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, 400, "bad request")

	if rec.Code != 400 {
		t.Errorf("expected status 400, got %d", rec.Code)
	}

	var resp ResponsePayload
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", resp.Status)
	}
	if resp.Message != "bad request" {
		t.Errorf("expected message 'bad request', got '%s'", resp.Message)
	}
	if resp.Data != nil {
		t.Errorf("expected data to be nil, got %v", resp.Data)
	}
}

func TestWriteError_NotFound(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, 404, "not found")

	if rec.Code != 404 {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var resp ResponsePayload
	json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got '%s'", resp.Status)
	}
}

func TestWriteError_InternalServerError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, 500, "internal error")

	if rec.Code != 500 {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestWriteJSON_SetsContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, 200, ResponsePayload{Status: "success", Message: "test"})

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestWriteJSON_ValidJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSON(rec, 200, ResponsePayload{Status: "success", Message: "hello"})

	var result map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("expected status 'success', got '%v'", result["status"])
	}
	if result["message"] != "hello" {
		t.Errorf("expected message 'hello', got '%v'", result["message"])
	}
}

func TestResponsePayload_OmitEmptyData(t *testing.T) {
	payload := ResponsePayload{
		Status:  "success",
		Message: "no data",
		Data:    nil,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(b, &result)

	if _, exists := result["data"]; exists {
		t.Error("expected 'data' field to be omitted when nil")
	}
}

func TestResponsePayload_IncludesData(t *testing.T) {
	payload := ResponsePayload{
		Status:  "success",
		Message: "has data",
		Data:    "something",
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(b, &result)

	if _, exists := result["data"]; !exists {
		t.Error("expected 'data' field to be present")
	}
}
