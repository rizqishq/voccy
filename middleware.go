package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/cors"
	"github.com/google/uuid"
)

type contextKey string

const (
	ctxFingerprint contextKey = "fingerprint"
	ctxOrgID       contextKey = "orgID"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), ctxOrgID, claims.OrgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetOrgIDFromContext(ctx context.Context) uuid.UUID {
	if orgID, ok := ctx.Value(ctxOrgID).(uuid.UUID); ok {
		return orgID
	}
	return uuid.Nil
}

func CORSMiddleware() func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
		MaxAge:         300,
	})
}

func PanicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type rateLimiterEntry struct {
	timestamps []time.Time
	mu         sync.Mutex
}

var (
	rateLimiterMap = sync.Map{}
	cleanupOnce    sync.Once
)

func RateLimiter(requestsPerMinute int) func(http.Handler) http.Handler {
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				cleanupOldEntries()
			}
		}()
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)
			now := time.Now()

			val, _ := rateLimiterMap.LoadOrStore(ip, &rateLimiterEntry{
				timestamps: make([]time.Time, 0),
			})
			entry := val.(*rateLimiterEntry)

			entry.mu.Lock()
			defer entry.mu.Unlock()

			cutoff := now.Add(-time.Minute)
			valid := entry.timestamps[:0]
			for _, ts := range entry.timestamps {
				if ts.After(cutoff) {
					valid = append(valid, ts)
				}
			}
			entry.timestamps = valid

			if len(entry.timestamps) >= requestsPerMinute {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}

			entry.timestamps = append(entry.timestamps, now)
			next.ServeHTTP(w, r)
		})
	}
}

func FingerprintMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		ua := r.Header.Get("User-Agent")

		hash := sha256.Sum256([]byte(ip + ua))
		fingerprint := hex.EncodeToString(hash[:])[:16]

		ctx := context.WithValue(r.Context(), ctxFingerprint, fingerprint)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetFingerprint(ctx context.Context) string {
	if fp, ok := ctx.Value(ctxFingerprint).(string); ok {
		return fp
	}
	return ""
}

func getClientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func cleanupOldEntries() {
	cutoff := time.Now().Add(-5 * time.Minute)
	rateLimiterMap.Range(func(key, value any) bool {
		entry := value.(*rateLimiterEntry)
		entry.mu.Lock()
		isEmpty := len(entry.timestamps) == 0 ||
			entry.timestamps[len(entry.timestamps)-1].Before(cutoff)
		entry.mu.Unlock()
		if isEmpty {
			rateLimiterMap.Delete(key)
		}
		return true
	})
}
