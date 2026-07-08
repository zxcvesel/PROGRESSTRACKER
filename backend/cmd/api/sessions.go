package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func createSessionHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	var request CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if request.StartedAt == "" || request.EndedAt == "" || request.DurationMinutes <= 0 {
		writeError(w, "startedAt, endedAt, and positive durationMinutes are required", http.StatusBadRequest)
		return
	}

	sessionDate := sessionDateString(request.EndedAt)
	createdAt := time.Now().Format(time.RFC3339)
	existingSession, hasExistingSession, err := loadSessionForDate(goal.ID, sessionDate)
	if err != nil {
		writeError(w, "failed to load daily session", http.StatusInternalServerError)
		return
	}

	if hasExistingSession {
		updatedNotes := mergeSessionNotes(existingSession.Notes, request.Notes)
		updatedTags := mergeTags(existingSession.Tags, request.Tags)
		_, err := db.Exec(`
			UPDATE sessions
			SET ended_at = ?, duration_minutes = ?, notes = ?, tags = ?
			WHERE id = ? AND goal_id = ?
		`, request.EndedAt, existingSession.DurationMinutes+request.DurationMinutes, updatedNotes, tagsToString(updatedTags), existingSession.ID, goal.ID)
		if err != nil {
			writeError(w, "failed to update daily session", http.StatusInternalServerError)
			return
		}

		session, err := loadSession(goal.ID, existingSession.ID)
		if err != nil {
			writeError(w, "failed to load daily session", http.StatusInternalServerError)
			return
		}

		if err := refreshDailyProgressForGoal(goal); err != nil {
			writeError(w, "failed to refresh daily progress", http.StatusInternalServerError)
			return
		}

		writeJSON(w, session, http.StatusOK)
		return
	}

	result, err := db.Exec(`
		INSERT INTO sessions (
			goal_id, started_at, ended_at, duration_minutes,
			notes, tags, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, goal.ID, request.StartedAt, request.EndedAt, request.DurationMinutes, request.Notes, tagsToString(request.Tags), createdAt)
	if err != nil {
		writeError(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, "failed to read created session", http.StatusInternalServerError)
		return
	}

	session := Session{
		ID:              int(id),
		GoalID:          goal.ID,
		StartedAt:       request.StartedAt,
		EndedAt:         request.EndedAt,
		DurationMinutes: request.DurationMinutes,
		Notes:           request.Notes,
		Tags:            cleanTags(request.Tags),
		CreatedAt:       createdAt,
	}

	if err := refreshDailyProgressForGoal(goal); err != nil {
		writeError(w, "failed to refresh daily progress", http.StatusInternalServerError)
		return
	}

	writeJSON(w, session, http.StatusCreated)
}

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
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, "invalid JSON", http.StatusBadRequest)
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
