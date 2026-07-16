package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

const (
	verificationTokenLifetime = 24 * time.Hour
	passwordResetLifetime     = time.Hour
)

func verifyEmailHandler(w http.ResponseWriter, r *http.Request) {
	var request ActionTokenRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	token := strings.TrimSpace(request.Token)
	userID, err := loadActionTokenUserID(token, "verify_email")
	if err == sql.ErrNoRows {
		writeError(w, "verification link is invalid or expired", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeError(w, "failed to verify email", http.StatusInternalServerError)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, "failed to verify email", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE users SET email_verified = 1 WHERE id = ?`, userID); err != nil {
		writeError(w, "failed to verify email", http.StatusInternalServerError)
		return
	}
	if ok, err := deleteActionToken(tx, token, "verify_email"); err != nil || !ok {
		writeError(w, "verification link is invalid or expired", http.StatusBadRequest)
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, "failed to verify email", http.StatusInternalServerError)
		return
	}
	user, err := loadUserByID(userID)
	if err != nil {
		writeError(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	writeJSON(w, AuthResponse{User: user}, http.StatusOK)
}

func resendVerificationHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}
	if user.EmailVerified {
		writeJSON(w, ActionResponse{Message: "email is already verified"}, http.StatusOK)
		return
	}
	token, err := issueActionToken(user.ID, "verify_email", verificationTokenLifetime)
	if err != nil {
		writeError(w, "failed to create verification link", http.StatusInternalServerError)
		return
	}
	if err := sendAccountActionEmail(user.Email, "verify_email", token); err != nil {
		writeError(w, "failed to send verification email", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, actionResponse("verification email sent", token), http.StatusOK)
}

func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if !authRateLimiter.Allow(rateLimitKey(r, "forgot-password")) {
		writeError(w, "too many password reset attempts", http.StatusTooManyRequests)
		return
	}
	var request ForgotPasswordRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	email, valid := normalizeAndValidateEmail(request.Email)
	response := ActionResponse{Message: "if the account exists, a reset email has been sent"}
	if !valid {
		writeJSON(w, response, http.StatusOK)
		return
	}
	user, err := loadUserByEmail(email)
	if err == sql.ErrNoRows {
		verifyPassword("invalid-password", dummyPasswordHash)
		writeJSON(w, response, http.StatusOK)
		return
	}
	if err != nil {
		writeError(w, "failed to prepare password reset", http.StatusInternalServerError)
		return
	}
	token, err := issueActionToken(user.ID, "reset_password", passwordResetLifetime)
	if err != nil {
		writeError(w, "failed to prepare password reset", http.StatusInternalServerError)
		return
	}
	if err := sendAccountActionEmail(user.Email, "reset_password", token); err != nil {
		writeError(w, "failed to send reset email", http.StatusServiceUnavailable)
		return
	}
	if developmentActionTokensEnabled() {
		response.DevelopmentToken = token
	}
	writeJSON(w, response, http.StatusOK)
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var request ResetPasswordRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if !validPasswordLength(request.NewPassword) || !isStrongPassword(request.NewPassword) {
		writeError(w, "password must be 8-128 characters and include uppercase letters, numbers, and special characters", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(request.Token)
	userID, err := loadActionTokenUserID(token, "reset_password")
	if err == sql.ErrNoRows {
		writeError(w, "reset link is invalid or expired", http.StatusBadRequest)
		return
	}
	if err != nil {
		writeError(w, "failed to reset password", http.StatusInternalServerError)
		return
	}
	hash, err := hashPassword(request.NewPassword)
	if err != nil {
		writeError(w, "failed to protect password", http.StatusInternalServerError)
		return
	}
	tx, err := db.Begin()
	if err != nil {
		writeError(w, "failed to reset password", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if ok, err := deleteActionToken(tx, token, "reset_password"); err != nil || !ok {
		writeError(w, "reset link is invalid or expired", http.StatusBadRequest)
		return
	}
	if _, err := tx.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, hash, userID); err != nil {
		writeError(w, "failed to reset password", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec(`DELETE FROM auth_sessions WHERE user_id = ?`, userID); err != nil {
		writeError(w, "failed to reset password", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, "failed to reset password", http.StatusInternalServerError)
		return
	}
	clearAuthCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func logoutAllHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}
	if _, err := db.Exec(`DELETE FROM auth_sessions WHERE user_id = ?`, user.ID); err != nil {
		writeError(w, "failed to close sessions", http.StatusInternalServerError)
		return
	}
	clearAuthCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}
	var request DeleteAccountRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	fullUser, err := loadUserByEmail(user.Email)
	if err != nil || !verifyPassword(request.Password, fullUser.PasswordHash) {
		writeError(w, "password is incorrect", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		writeError(w, "failed to delete account", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM sessions WHERE goal_id IN (SELECT id FROM goals WHERE user_id = ?)`, user.ID); err != nil {
		writeError(w, "failed to delete account", http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec(`DELETE FROM daily_progress WHERE goal_id IN (SELECT id FROM goals WHERE user_id = ?)`, user.ID); err != nil {
		writeError(w, "failed to delete account", http.StatusInternalServerError)
		return
	}
	for _, query := range []string{
		`DELETE FROM active_timers WHERE user_id = ?`,
		`DELETE FROM goals WHERE user_id = ?`,
		`DELETE FROM entries WHERE user_id = ?`,
		`DELETE FROM action_tokens WHERE user_id = ?`,
		`DELETE FROM auth_sessions WHERE user_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.Exec(query, user.ID); err != nil {
			writeError(w, "failed to delete account", http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, "failed to delete account", http.StatusInternalServerError)
		return
	}
	clearAuthCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func issueActionToken(userID int, kind string, lifetime time.Duration) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	now := time.Now()
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM action_tokens WHERE user_id = ? AND kind = ?`, userID, kind); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`
		INSERT INTO action_tokens (token_hash, user_id, kind, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, tokenHash(token), userID, kind, now.Format(time.RFC3339), now.Add(lifetime).Format(time.RFC3339)); err != nil {
		return "", err
	}
	return token, tx.Commit()
}

func loadActionTokenUserID(token string, kind string) (int, error) {
	if token == "" {
		return 0, sql.ErrNoRows
	}
	var userID int
	err := db.QueryRow(`
		SELECT user_id FROM action_tokens
		WHERE token_hash = ? AND kind = ? AND expires_at > ?
	`, tokenHash(token), kind, time.Now().Format(time.RFC3339)).Scan(&userID)
	return userID, err
}

type actionTokenExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func deleteActionToken(executor actionTokenExecutor, token string, kind string) (bool, error) {
	result, err := executor.Exec(`
		DELETE FROM action_tokens
		WHERE token_hash = ? AND kind = ? AND expires_at > ?
	`, tokenHash(token), kind, time.Now().Format(time.RFC3339))
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func actionResponse(message string, token string) ActionResponse {
	response := ActionResponse{Message: message}
	if developmentActionTokensEnabled() {
		response.DevelopmentToken = token
	}
	return response
}
