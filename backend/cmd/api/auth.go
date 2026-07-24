package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if !authRateLimiter.Allow(rateLimitKey(r, "register")) {
		writeError(w, "too many registration attempts", http.StatusTooManyRequests)
		return
	}

	var request AuthRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	email, validEmail := normalizeAndValidateEmail(request.Email)
	name := strings.TrimSpace(request.Name)
	if !validEmail {
		writeError(w, "valid email is required", http.StatusBadRequest)
		return
	}
	if !validTextLength(name, maxNameLength) {
		writeError(w, "name must not exceed 100 characters", http.StatusBadRequest)
		return
	}
	if !validPasswordLength(request.Password) || !isStrongPassword(request.Password) {
		writeError(w, "password must be 8-128 characters and include uppercase letters, numbers, and special characters", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(request.Password)
	if err != nil {
		writeError(w, "failed to protect password", http.StatusInternalServerError)
		return
	}
	timezone, validTimezone := normalizeTimezone(request.Timezone)
	if !validTimezone {
		writeError(w, "invalid timezone", http.StatusBadRequest)
		return
	}

	createdAt := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO users (email, name, password_hash, created_at, email_verified, timezone)
		VALUES (?, ?, ?, ?, 0, ?)
	`, email, name, passwordHash, createdAt, timezone)
	if err != nil {
		writeError(w, "user already exists", http.StatusConflict)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, "failed to read created user", http.StatusInternalServerError)
		return
	}

	user := User{
		ID:            int(id),
		Email:         email,
		Name:          name,
		CreatedAt:     createdAt,
		EmailVerified: false,
		Timezone:      timezone,
	}
	token, err := createAuthSession(user.ID)
	if err != nil {
		cleanupIncompleteRegistration(user.ID)
		writeError(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, token)
	verificationToken, err := issueActionToken(user.ID, "verify_email", verificationTokenLifetime)
	if err != nil {
		cleanupIncompleteRegistration(user.ID)
		clearAuthCookie(w)
		writeError(w, "failed to create verification link", http.StatusInternalServerError)
		return
	}
	if err := sendAccountActionEmail(user.Email, "verify_email", verificationToken); err != nil {
		cleanupIncompleteRegistration(user.ID)
		clearAuthCookie(w)
		writeError(w, "failed to send verification email", http.StatusServiceUnavailable)
		return
	}

	response := AuthResponse{User: user}
	if developmentActionTokensEnabled() {
		response.DevelopmentToken = verificationToken
	}
	writeJSON(w, response, http.StatusCreated)
}

func cleanupIncompleteRegistration(userID int) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	for _, query := range []string{
		`DELETE FROM action_tokens WHERE user_id = ?`,
		`DELETE FROM auth_sessions WHERE user_id = ?`,
		`DELETE FROM users WHERE id = ?`,
	} {
		if _, err := tx.Exec(query, userID); err != nil {
			return
		}
	}
	_ = tx.Commit()
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var request AuthRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	email := normalizeEmail(request.Email)
	retryAfter, err := loginAttempts.retryAfter(r, email)
	if err != nil {
		writeError(w, "failed to check login attempts", http.StatusInternalServerError)
		return
	}
	if retryAfter > 0 {
		writeLoginRateLimit(w, retryAfter)
		return
	}

	email, validEmail := normalizeAndValidateEmail(email)
	if !validEmail || len(request.Password) > maxPasswordLength {
		rejectInvalidLogin(w, r, email)
		return
	}

	user, err := loadUserByEmail(email)
	if err == sql.ErrNoRows {
		verifyPassword(request.Password, dummyPasswordHash)
		rejectInvalidLogin(w, r, email)
		return
	}
	if err != nil {
		writeError(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if !verifyPassword(request.Password, user.PasswordHash) {
		rejectInvalidLogin(w, r, email)
		return
	}
	if err := loginAttempts.clearAccountFailures(email); err != nil {
		writeError(w, "failed to clear login attempts", http.StatusInternalServerError)
		return
	}
	if passwordNeedsRehash(user.PasswordHash) {
		passwordHash, err := hashPassword(request.Password)
		if err == nil {
			_, _ = db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, user.ID)
		}
	}

	user.PasswordHash = ""
	token, err := createAuthSession(user.ID)
	if err != nil {
		writeError(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, token)

	writeJSON(w, AuthResponse{User: user}, http.StatusOK)
}

func rejectInvalidLogin(w http.ResponseWriter, r *http.Request, email string) {
	retryAfter, err := loginAttempts.recordFailure(r, email)
	if err != nil {
		writeError(w, "failed to record login attempt", http.StatusInternalServerError)
		return
	}
	if retryAfter > 0 {
		writeLoginRateLimit(w, retryAfter)
		return
	}
	writeError(w, "invalid email or password", http.StatusUnauthorized)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	token, ok := bearerToken(r)
	if ok {
		_, _ = db.Exec(`DELETE FROM auth_sessions WHERE token_hash = ?`, tokenHash(token))
	}
	clearAuthCookie(w)

	w.WriteHeader(http.StatusNoContent)
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	writeJSON(w, user, http.StatusOK)
}

func updateMeHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	var request UpdateProfileRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	name := strings.TrimSpace(request.Name)
	if !validTextLength(name, maxNameLength) {
		writeError(w, "name must not exceed 100 characters", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`UPDATE users SET name = ? WHERE id = ?`, name, user.ID)
	if err != nil {
		writeError(w, "failed to update profile", http.StatusInternalServerError)
		return
	}

	user.Name = name
	writeJSON(w, user, http.StatusOK)
}

func changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	var request ChangePasswordRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if len(request.CurrentPassword) > maxPasswordLength {
		writeError(w, "current password is incorrect", http.StatusBadRequest)
		return
	}

	fullUser, err := loadUserByEmail(user.Email)
	if err != nil {
		writeError(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if !verifyPassword(request.CurrentPassword, fullUser.PasswordHash) {
		writeError(w, "current password is incorrect", http.StatusBadRequest)
		return
	}
	if !validPasswordLength(request.NewPassword) || !isStrongPassword(request.NewPassword) {
		writeError(w, "password must be 8-128 characters and include uppercase letters, numbers, and special characters", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(request.NewPassword)
	if err != nil {
		writeError(w, "failed to protect password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, user.ID)
	if err != nil {
		writeError(w, "failed to change password", http.StatusInternalServerError)
		return
	}

	_, _ = db.Exec(`DELETE FROM auth_sessions WHERE user_id = ? AND token_hash != ?`, user.ID, tokenHashFromRequest(r))
	w.WriteHeader(http.StatusNoContent)
}

func currentUserFromRequest(w http.ResponseWriter, r *http.Request) (User, bool) {
	token, ok := bearerToken(r)
	if !ok {
		writeError(w, "authorization token is required", http.StatusUnauthorized)
		return User{}, false
	}

	user, err := loadUserByToken(token)
	if err == sql.ErrNoRows {
		writeError(w, "invalid or expired token", http.StatusUnauthorized)
		return User{}, false
	}
	if err != nil {
		writeError(w, "failed to read session", http.StatusInternalServerError)
		return User{}, false
	}

	return user, true
}

func currentVerifiedUserFromRequest(w http.ResponseWriter, r *http.Request) (User, bool) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return User{}, false
	}
	if !user.EmailVerified {
		writeError(w, "email verification is required", http.StatusForbidden)
		return User{}, false
	}
	return user, true
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token != "" {
			return token, true
		}
	}

	cookie, err := r.Cookie(authCookieName)
	if err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value), true
	}

	return "", false
}

func loadUserByEmail(email string) (User, error) {
	row := db.QueryRow(`
		SELECT id, email, name, password_hash, created_at, email_verified, timezone
		FROM users
		WHERE email = ?
	`, email)

	return scanUser(row)
}

func loadUserByID(id int) (User, error) {
	row := db.QueryRow(`
		SELECT id, email, name, password_hash, created_at, email_verified, timezone
		FROM users
		WHERE id = ?
	`, id)
	return scanUser(row)
}

func loadUserByToken(token string) (User, error) {
	row := db.QueryRow(`
		SELECT users.id, users.email, users.name, users.password_hash, users.created_at, users.email_verified, users.timezone
		FROM auth_sessions
		INNER JOIN users ON users.id = auth_sessions.user_id
		WHERE auth_sessions.token_hash = ? AND auth_sessions.expires_at > ?
	`, tokenHash(token), time.Now().Format(time.RFC3339))

	user, err := scanUser(row)
	if err != nil {
		return User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (User, error) {
	var user User
	err := scanner.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.EmailVerified,
		&user.Timezone,
	)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func createAuthSession(userID int) (string, error) {
	if err := cleanupExpiredAuthSessions(db); err != nil {
		return "", err
	}
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}

	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO auth_sessions (token_hash, user_id, created_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, tokenHash(token), userID, now.Format(time.RFC3339), now.Add(authTokenLifetime).Format(time.RFC3339))
	if err != nil {
		return "", err
	}

	return token, nil
}

func setAuthCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookiesEnabled(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(authTokenLifetime),
		MaxAge:   int(authTokenLifetime.Seconds()),
	})
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secureCookiesEnabled(),
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
