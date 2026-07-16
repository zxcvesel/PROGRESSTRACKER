package main

import (
	"net/http"
	"strings"
)

func updateSessionHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	sessionID, ok := sessionIDFromRequest(w, r)
	if !ok {
		return
	}

	var request UpdateSessionRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Notes = strings.TrimSpace(request.Notes)
	request.Tags = cleanTags(request.Tags)
	if message := validateSessionContent(request.Notes, request.Tags); message != "" {
		writeError(w, message, http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		UPDATE sessions
		SET notes = ?, tags = ?
		WHERE id = ? AND goal_id = ?
	`, request.Notes, tagsToString(request.Tags), sessionID, goal.ID)
	if err != nil {
		writeError(w, "failed to update session", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		writeError(w, "failed to read updated session", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		writeError(w, "session not found", http.StatusNotFound)
		return
	}

	session, err := loadSession(goal.ID, sessionID)
	if err != nil {
		writeError(w, "failed to load updated session", http.StatusInternalServerError)
		return
	}

	writeJSON(w, session, http.StatusOK)
}

func deleteSessionHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	sessionID, ok := sessionIDFromRequest(w, r)
	if !ok {
		return
	}

	result, err := db.Exec(`DELETE FROM sessions WHERE id = ? AND goal_id = ?`, sessionID, goal.ID)
	if err != nil {
		writeError(w, "failed to delete session", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		writeError(w, "failed to read deleted session", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		writeError(w, "session not found", http.StatusNotFound)
		return
	}

	if err := refreshDailyProgressForGoal(goal); err != nil {
		writeError(w, "failed to refresh daily progress", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
