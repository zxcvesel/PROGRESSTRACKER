package main

import (
	"net/http"
	"time"
)

func goalsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}

	goals, err := loadGoals(user.ID)
	if err != nil {
		writeError(w, "failed to load goals", http.StatusInternalServerError)
		return
	}

	summaries := make([]GoalSummary, 0, len(goals))
	for _, goal := range goals {
		summary, err := buildGoalSummary(goal)
		if err != nil {
			writeError(w, "failed to load goal summary", http.StatusInternalServerError)
			return
		}
		summaries = append(summaries, summary)
	}

	writeJSON(w, summaries, http.StatusOK)
}

func createGoalHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}

	var request CreateGoalRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	if message := validateGoalInput(&request.Title, &request.Description, request.TotalDays, request.DailyTargetMinutes); message != "" {
		writeError(w, message, http.StatusBadRequest)
		return
	}

	if len(request.ActiveWeekdays) == 0 {
		request.ActiveWeekdays = []int{1, 2, 3, 4, 5, 6, 7}
	}
	if !validateWeekdays(request.ActiveWeekdays) {
		writeError(w, "activeWeekdays must contain unique values from 1 to 7", http.StatusBadRequest)
		return
	}

	if request.StartDate == "" {
		request.StartDate = todayStringForUser(user.ID)
	}
	if _, err := time.Parse(time.DateOnly, request.StartDate); err != nil {
		writeError(w, "startDate must use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	createdAt := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO goals (
			title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status, user_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?)
	`, request.Title, request.Description, request.TotalDays, request.DailyTargetMinutes, weekdaysToString(request.ActiveWeekdays), request.StartDate, createdAt, user.ID)
	if err != nil {
		writeError(w, "failed to create goal", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, "failed to read created goal", http.StatusInternalServerError)
		return
	}

	goal := Goal{
		ID:                 int(id),
		UserID:             user.ID,
		Title:              request.Title,
		Description:        request.Description,
		TotalDays:          request.TotalDays,
		DailyTargetMinutes: request.DailyTargetMinutes,
		ActiveWeekdays:     request.ActiveWeekdays,
		StartDate:          request.StartDate,
		CreatedAt:          createdAt,
		Status:             "active",
	}

	if err := refreshDailyProgressForGoal(goal); err != nil {
		writeError(w, "failed to initialize goal progress", http.StatusInternalServerError)
		return
	}

	summary, err := buildGoalSummary(goal)
	if err != nil {
		writeError(w, "failed to load created goal", http.StatusInternalServerError)
		return
	}

	writeJSON(w, summary, http.StatusCreated)
}

func goalDetailHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	summary, err := buildGoalSummary(goal)
	if err != nil {
		writeError(w, "failed to load goal summary", http.StatusInternalServerError)
		return
	}

	sessions, err := loadSessions(goal.ID, 12)
	if err != nil {
		writeError(w, "failed to load sessions", http.StatusInternalServerError)
		return
	}

	calendar, err := loadCalendarStats(goal.ID, goal.UserID, 42)
	if err != nil {
		writeError(w, "failed to load calendar", http.StatusInternalServerError)
		return
	}

	detail := GoalDetail{
		GoalSummary:           summary,
		TodayRemainingMinutes: max(goal.DailyTargetMinutes-summary.TodayMinutes, 0),
		RecentSessions:        sessions,
		Calendar:              calendar,
	}

	writeJSON(w, detail, http.StatusOK)
}

func updateGoalHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	var request UpdateGoalRequest
	if !decodeJSON(w, r, &request) {
		return
	}

	if message := validateGoalInput(&request.Title, &request.Description, request.TotalDays, request.DailyTargetMinutes); message != "" {
		writeError(w, message, http.StatusBadRequest)
		return
	}

	if request.Status == "" {
		request.Status = goal.Status
	}
	if request.Status != "active" && request.Status != "completed" {
		writeError(w, "status must be active or completed", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		UPDATE goals
		SET title = ?, description = ?, total_days = ?, daily_target_minutes = ?, status = ?
		WHERE id = ?
	`, request.Title, request.Description, request.TotalDays, request.DailyTargetMinutes, request.Status, goal.ID)
	if err != nil {
		writeError(w, "failed to update goal", http.StatusInternalServerError)
		return
	}

	updatedGoal, err := loadGoal(goal.ID, goal.UserID)
	if err != nil {
		writeError(w, "failed to load updated goal", http.StatusInternalServerError)
		return
	}

	if err := refreshDailyProgressForGoal(updatedGoal); err != nil {
		writeError(w, "failed to refresh goal progress", http.StatusInternalServerError)
		return
	}

	summary, err := buildGoalSummary(updatedGoal)
	if err != nil {
		writeError(w, "failed to load updated goal summary", http.StatusInternalServerError)
		return
	}

	writeJSON(w, summary, http.StatusOK)
}

func deleteGoalHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		writeError(w, "failed to start delete", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions WHERE goal_id = ?`, goal.ID); err != nil {
		writeError(w, "failed to delete goal sessions", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(`DELETE FROM daily_progress WHERE goal_id = ?`, goal.ID); err != nil {
		writeError(w, "failed to delete goal progress", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(`DELETE FROM goals WHERE id = ?`, goal.ID); err != nil {
		writeError(w, "failed to delete goal", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, "failed to finish delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
