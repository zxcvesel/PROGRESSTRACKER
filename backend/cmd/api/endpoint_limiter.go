package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	registrationRequestLimit   = 10
	registrationRequestWindow  = time.Hour
	passwordResetClientLimit   = 10
	passwordResetAccountLimit  = 5
	passwordResetRequestWindow = time.Hour
	verificationClientLimit    = 10
	verificationAccountLimit   = 5
	verificationRequestWindow  = time.Hour
	endpointRateLimitRetention = 7 * 24 * time.Hour
)

type endpointRateRule struct {
	action string
	scope  string
	value  string
	limit  int
	window time.Duration
}

type endpointRateLimiter struct {
	now func() time.Time
}

type endpointRateState struct {
	count         int
	windowStarted time.Time
}

func newEndpointRateLimiter() *endpointRateLimiter {
	return &endpointRateLimiter{now: time.Now}
}

func (limiter *endpointRateLimiter) consume(rules ...endpointRateRule) (time.Duration, error) {
	now := limiter.now().UTC()
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM endpoint_rate_limits WHERE updated_at < ?`,
		now.Add(-endpointRateLimitRetention).Format(time.RFC3339Nano)); err != nil {
		return 0, err
	}

	type pendingRateLimit struct {
		key   string
		rule  endpointRateRule
		state endpointRateState
	}
	pending := make([]pendingRateLimit, 0, len(rules))
	longestRetry := time.Duration(0)

	for _, rule := range rules {
		if rule.value == "" || rule.limit <= 0 || rule.window <= 0 {
			continue
		}
		key := endpointRateLimitKey(rule.action, rule.scope, rule.value)
		state, found, err := loadEndpointRateState(tx, key)
		if err != nil {
			return 0, err
		}
		if !found || now.Sub(state.windowStarted) >= rule.window {
			state = endpointRateState{windowStarted: now}
		}
		if state.count >= rule.limit {
			longestRetry = longerDuration(longestRetry, state.windowStarted.Add(rule.window).Sub(now))
		}
		pending = append(pending, pendingRateLimit{key: key, rule: rule, state: state})
	}

	if longestRetry > 0 {
		return longestRetry, nil
	}

	for _, item := range pending {
		item.state.count++
		if _, err := tx.Exec(`
			INSERT INTO endpoint_rate_limits (
				key_hash, request_count, window_started_at, updated_at
			)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(key_hash) DO UPDATE SET
				request_count = excluded.request_count,
				window_started_at = excluded.window_started_at,
				updated_at = excluded.updated_at
		`, item.key, item.state.count, item.state.windowStarted.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano)); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return 0, nil
}

func registrationRateRules(r *http.Request) []endpointRateRule {
	return []endpointRateRule{{
		action: "register",
		scope:  "client",
		value:  clientAddress(r),
		limit:  registrationRequestLimit,
		window: registrationRequestWindow,
	}}
}

func passwordResetClientRateRules(r *http.Request) []endpointRateRule {
	return []endpointRateRule{{
		action: "forgot-password",
		scope:  "client",
		value:  clientAddress(r),
		limit:  passwordResetClientLimit,
		window: passwordResetRequestWindow,
	}}
}

func passwordResetAccountRateRules(email string) []endpointRateRule {
	return []endpointRateRule{{
		action: "forgot-password",
		scope:  "account",
		value:  normalizeEmail(email),
		limit:  passwordResetAccountLimit,
		window: passwordResetRequestWindow,
	}}
}

func verificationRateRules(r *http.Request, userID int) []endpointRateRule {
	return []endpointRateRule{
		{
			action: "resend-verification",
			scope:  "client",
			value:  clientAddress(r),
			limit:  verificationClientLimit,
			window: verificationRequestWindow,
		},
		{
			action: "resend-verification",
			scope:  "account",
			value:  strconv.Itoa(userID),
			limit:  verificationAccountLimit,
			window: verificationRequestWindow,
		},
	}
}

func endpointRateLimitKey(action string, scope string, value string) string {
	return tokenHash("endpoint-rate:" + action + ":" + scope + ":" + value)
}

type endpointRateQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func loadEndpointRateState(querier endpointRateQuerier, key string) (endpointRateState, bool, error) {
	var state endpointRateState
	var windowStarted string
	err := querier.QueryRow(`
		SELECT request_count, window_started_at
		FROM endpoint_rate_limits
		WHERE key_hash = ?
	`, key).Scan(&state.count, &windowStarted)
	if err == sql.ErrNoRows {
		return endpointRateState{}, false, nil
	}
	if err != nil {
		return endpointRateState{}, false, err
	}
	state.windowStarted, err = time.Parse(time.RFC3339Nano, windowStarted)
	if err != nil {
		return endpointRateState{}, false, fmt.Errorf("parse endpoint rate limit window: %w", err)
	}
	return state, true, nil
}

func enforceEndpointRateLimit(w http.ResponseWriter, message string, rules ...endpointRateRule) bool {
	retryAfter, err := endpointLimits.consume(rules...)
	if err != nil {
		writeError(w, "failed to check request limit", http.StatusInternalServerError)
		return false
	}
	if retryAfter <= 0 {
		return true
	}
	seconds := max(1, int((retryAfter+time.Second-1)/time.Second))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeError(w, message, http.StatusTooManyRequests)
	return false
}
