package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func writeJSON(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, ErrorResponse{Error: message}, status)
}

type rateLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	maxKeys     int
	lastCleanup time.Time
	attempts    map[string]rateLimitBucket
}

type rateLimitBucket struct {
	times    []time.Time
	lastSeen time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:    limit,
		window:   window,
		maxKeys:  10000,
		attempts: map[string]rateLimitBucket{},
	}
}

func (limiter *rateLimiter) Allow(key string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-limiter.window)
	if limiter.lastCleanup.IsZero() || now.Sub(limiter.lastCleanup) >= limiter.window {
		limiter.cleanup(cutoff)
		limiter.lastCleanup = now
	}

	bucket := limiter.attempts[key]
	filtered := bucket.times[:0]
	for _, attempt := range bucket.times {
		if attempt.After(cutoff) {
			filtered = append(filtered, attempt)
		}
	}

	if len(filtered) >= limiter.limit {
		limiter.attempts[key] = rateLimitBucket{times: filtered, lastSeen: now}
		return false
	}
	if len(limiter.attempts) >= limiter.maxKeys {
		if _, exists := limiter.attempts[key]; !exists {
			return false
		}
	}

	limiter.attempts[key] = rateLimitBucket{times: append(filtered, now), lastSeen: now}
	return true
}

func (limiter *rateLimiter) cleanup(cutoff time.Time) {
	for key, bucket := range limiter.attempts {
		if bucket.lastSeen.Before(cutoff) {
			delete(limiter.attempts, key)
		}
	}
}

func rateLimitKey(r *http.Request, action string) string {
	return action + ":" + clientAddress(r)
}

func clientAddress(r *http.Request) string {
	remote := strings.TrimSpace(r.RemoteAddr)
	if strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_TRUST_PROXY")), "true") {
		if realIP := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); realIP != nil {
			remote = realIP.String()
		}
	}
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	if remote == "" {
		remote = "unknown"
	}
	return remote
}

func addColumnIfMissing(database *sql.DB, table string, column string, definition string) error {
	rows, err := database.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = database.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition))
	return err
}
