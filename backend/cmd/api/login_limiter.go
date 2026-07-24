package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	accountLoginFailureLimit = 8
	clientLoginFailureLimit  = 30
	loginFailureWindow       = 15 * time.Minute
	loginBlockDuration       = 15 * time.Minute
)

type loginAttemptPolicy struct {
	limit int
}

type loginAttemptLimiter struct {
	now           func() time.Time
	accountPolicy loginAttemptPolicy
	clientPolicy  loginAttemptPolicy
	window        time.Duration
	blockDuration time.Duration
}

type loginAttemptState struct {
	failures      int
	windowStarted time.Time
	blockedUntil  time.Time
}

func newLoginAttemptLimiter() *loginAttemptLimiter {
	return &loginAttemptLimiter{
		now:           time.Now,
		accountPolicy: loginAttemptPolicy{limit: accountLoginFailureLimit},
		clientPolicy:  loginAttemptPolicy{limit: clientLoginFailureLimit},
		window:        loginFailureWindow,
		blockDuration: loginBlockDuration,
	}
}

func (limiter *loginAttemptLimiter) retryAfter(r *http.Request, email string) (time.Duration, error) {
	now := limiter.now().UTC()
	keys := limiter.keys(r, email)
	longest := time.Duration(0)
	for _, key := range keys {
		state, found, err := loadLoginAttemptState(db, key.hash)
		if err != nil {
			return 0, err
		}
		if !found {
			continue
		}
		if state.blockedUntil.After(now) {
			longest = longerDuration(longest, state.blockedUntil.Sub(now))
		}
	}
	return longest, nil
}

func (limiter *loginAttemptLimiter) recordFailure(r *http.Request, email string) (time.Duration, error) {
	now := limiter.now().UTC()
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM login_attempts WHERE updated_at < ?`,
		now.Add(-24*time.Hour).Format(time.RFC3339Nano)); err != nil {
		return 0, err
	}

	longest := time.Duration(0)
	for _, key := range limiter.keys(r, email) {
		state, found, err := loadLoginAttemptState(tx, key.hash)
		if err != nil {
			return 0, err
		}
		if !found || !state.blockedUntil.After(now) && now.Sub(state.windowStarted) >= limiter.window {
			state = loginAttemptState{windowStarted: now}
		}
		state.failures++
		if state.failures >= key.policy.limit {
			state.blockedUntil = now.Add(limiter.blockDuration)
			longest = longerDuration(longest, limiter.blockDuration)
		}
		if _, err := tx.Exec(`
			INSERT INTO login_attempts (
				key_hash, failure_count, window_started_at, blocked_until, updated_at
			)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(key_hash) DO UPDATE SET
				failure_count = excluded.failure_count,
				window_started_at = excluded.window_started_at,
				blocked_until = excluded.blocked_until,
				updated_at = excluded.updated_at
		`, key.hash, state.failures, state.windowStarted.Format(time.RFC3339Nano),
			formatOptionalTime(state.blockedUntil), now.Format(time.RFC3339Nano)); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return longest, nil
}

func (limiter *loginAttemptLimiter) clearAccountFailures(email string) error {
	_, err := db.Exec(`DELETE FROM login_attempts WHERE key_hash = ?`, loginAttemptKey("account", normalizeEmail(email)))
	return err
}

type scopedLoginAttemptKey struct {
	hash   string
	policy loginAttemptPolicy
}

func (limiter *loginAttemptLimiter) keys(r *http.Request, email string) []scopedLoginAttemptKey {
	return []scopedLoginAttemptKey{
		{hash: loginAttemptKey("account", normalizeEmail(email)), policy: limiter.accountPolicy},
		{hash: loginAttemptKey("client", clientAddress(r)), policy: limiter.clientPolicy},
	}
}

func loginAttemptKey(scope string, value string) string {
	return tokenHash("login-attempt:" + scope + ":" + value)
}

type loginAttemptQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func loadLoginAttemptState(querier loginAttemptQuerier, keyHash string) (loginAttemptState, bool, error) {
	var state loginAttemptState
	var windowStarted string
	var blockedUntil string
	err := querier.QueryRow(`
		SELECT failure_count, window_started_at, blocked_until
		FROM login_attempts
		WHERE key_hash = ?
	`, keyHash).Scan(&state.failures, &windowStarted, &blockedUntil)
	if err == sql.ErrNoRows {
		return loginAttemptState{}, false, nil
	}
	if err != nil {
		return loginAttemptState{}, false, err
	}
	state.windowStarted, err = time.Parse(time.RFC3339Nano, windowStarted)
	if err != nil {
		return loginAttemptState{}, false, fmt.Errorf("parse login attempt window: %w", err)
	}
	if blockedUntil != "" {
		state.blockedUntil, err = time.Parse(time.RFC3339Nano, blockedUntil)
		if err != nil {
			return loginAttemptState{}, false, fmt.Errorf("parse login attempt block: %w", err)
		}
	}
	return state, true, nil
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func writeLoginRateLimit(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := max(1, int((retryAfter+time.Second-1)/time.Second))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, "too many login attempts; try again later", http.StatusTooManyRequests)
}

func longerDuration(left time.Duration, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}
