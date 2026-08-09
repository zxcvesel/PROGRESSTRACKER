package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	userRoleUser  = "user"
	userRoleAdmin = "admin"
)

func configuredAdminEmail() (string, error) {
	value := strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_ADMIN_EMAIL"))
	if value == "" {
		return "", nil
	}

	email, valid := normalizeAndValidateEmail(value)
	if !valid {
		return "", fmt.Errorf("PROGRESS_TRACKER_ADMIN_EMAIL must be a valid email")
	}
	return email, nil
}

func roleForEmail(email string) string {
	adminEmail, err := configuredAdminEmail()
	if err == nil && adminEmail != "" && normalizeEmail(email) == adminEmail {
		return userRoleAdmin
	}
	return userRoleUser
}

func syncConfiguredAdminRole(database *sql.DB) error {
	adminEmail, err := configuredAdminEmail()
	if err != nil {
		return err
	}
	if adminEmail == "" {
		return nil
	}

	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin administrator configuration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE users SET role = ? WHERE role = ? AND email != ?`, userRoleUser, userRoleAdmin, adminEmail); err != nil {
		return fmt.Errorf("remove previous administrator: %w", err)
	}
	if _, err := tx.Exec(`UPDATE users SET role = ? WHERE email = ?`, userRoleAdmin, adminEmail); err != nil {
		return fmt.Errorf("configure administrator: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit administrator configuration: %w", err)
	}
	return nil
}

func currentAdminFromRequest(w http.ResponseWriter, r *http.Request) (User, bool) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return User{}, false
	}
	if user.Role != userRoleAdmin {
		writeError(w, "administrator access is required", http.StatusForbidden)
		return User{}, false
	}
	return user, true
}

func adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	if _, ok := currentAdminFromRequest(w, r); !ok {
		return
	}

	rows, err := db.Query(`
		SELECT users.id, users.email, users.name, users.created_at,
			users.email_verified, users.role,
			(SELECT COUNT(*) FROM auth_sessions
				WHERE auth_sessions.user_id = users.id AND auth_sessions.expires_at > ?),
			(SELECT COUNT(*) FROM goals WHERE goals.user_id = users.id)
		FROM users
		ORDER BY users.created_at ASC, users.id ASC
	`, time.Now().Format(time.RFC3339))
	if err != nil {
		writeError(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []AdminUserSummary{}
	for rows.Next() {
		var user AdminUserSummary
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.CreatedAt,
			&user.EmailVerified,
			&user.Role,
			&user.ActiveSessions,
			&user.GoalCount,
		); err != nil {
			writeError(w, "failed to read users", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		writeError(w, "failed to read users", http.StatusInternalServerError)
		return
	}

	writeJSON(w, users, http.StatusOK)
}

func revokeUserSessionsHandler(w http.ResponseWriter, r *http.Request) {
	admin, ok := currentAdminFromRequest(w, r)
	if !ok {
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		writeError(w, "invalid user id", http.StatusBadRequest)
		return
	}
	if userID == admin.ID {
		writeError(w, "cannot revoke the current administrator session", http.StatusBadRequest)
		return
	}

	var exists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, userID).Scan(&exists); err != nil {
		writeError(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if exists == 0 {
		writeError(w, "user not found", http.StatusNotFound)
		return
	}

	result, err := db.Exec(`DELETE FROM auth_sessions WHERE user_id = ?`, userID)
	if err != nil {
		writeError(w, "failed to revoke sessions", http.StatusInternalServerError)
		return
	}
	revoked, err := result.RowsAffected()
	if err != nil {
		writeError(w, "failed to read revoked sessions", http.StatusInternalServerError)
		return
	}

	writeJSON(w, RevokeSessionsResponse{RevokedSessions: revoked}, http.StatusOK)
}
