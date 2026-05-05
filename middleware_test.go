package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	handler := RateLimiter(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	handler := RateLimiter(3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		handler.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	var resp ResponsePayload
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Message != "rate limit exceeded" {
		t.Errorf("expected 'rate limit exceeded', got '%s'", resp.Message)
	}
}

func TestRateLimiter_SeparateByIP(t *testing.T) {
	handler := RateLimiter(2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.1.1.1:1000"
		handler.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "2.2.2.2:2000"
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("different IP should not be rate limited, got %d", rec.Code)
	}
}

func TestPanicRecovery_HandlesPanic(t *testing.T) {
	handler := PanicRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	var resp ResponsePayload
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Message != "internal server error" {
		t.Errorf("expected 'internal server error', got '%s'", resp.Message)
	}
}

func TestPanicRecovery_NoPanic(t *testing.T) {
	handler := PanicRecovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestFingerprintMiddleware_GeneratesFingerprint(t *testing.T) {
	var captured string
	handler := FingerprintMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetFingerprint(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:5000"
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	handler.ServeHTTP(rec, req)

	if captured == "" {
		t.Fatal("expected fingerprint to be set")
	}
	if len(captured) != 16 {
		t.Errorf("expected fingerprint length 16, got %d", len(captured))
	}
}

func TestFingerprintMiddleware_SameInputSameOutput(t *testing.T) {
	var first, second string

	handler := FingerprintMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := GetFingerprint(r.Context())
		if first == "" {
			first = fp
		} else {
			second = fp
		}
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.5:8080"
		req.Header.Set("User-Agent", "ConsistentAgent")
		handler.ServeHTTP(rec, req)
	}

	if first != second {
		t.Errorf("same input should produce same fingerprint: %s != %s", first, second)
	}
}

func TestFingerprintMiddleware_DifferentInputDifferentOutput(t *testing.T) {
	var fp1, fp2 string

	handler := FingerprintMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := GetFingerprint(r.Context())
		if fp1 == "" {
			fp1 = fp
		} else {
			fp2 = fp
		}
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.1.1.1:100"
	req.Header.Set("User-Agent", "AgentA")
	handler.ServeHTTP(rec, req)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "2.2.2.2:200"
	req.Header.Set("User-Agent", "AgentB")
	handler.ServeHTTP(rec, req)

	if fp1 == fp2 {
		t.Error("different input should produce different fingerprint")
	}
}

func TestCORSMiddleware_Headers(t *testing.T) {
	handler := CORSMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Allow-Origin '*', got '%s'", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
