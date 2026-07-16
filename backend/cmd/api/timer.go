package main

import (
	"database/sql"
	"math"
	"net/http"
	"strings"
	"time"
)

type activeTimerRecord struct {
	ID                 int
	UserID             int
	GoalID             int
	SessionDate        string
	State              string
	StartedAt          string
	LastResumedAt      string
	AccumulatedSeconds float64
	TargetSeconds      int
	SpeedMultiplier    float64
}

func activeTimerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}

	record, found, err := loadActiveTimer(user.ID)
	if err != nil {
		writeError(w, "failed to load active timer", http.StatusInternalServerError)
		return
	}
	if !found {
		writeJSON(w, TimerStatusResponse{Active: false}, http.StatusOK)
		return
	}

	state, err := currentTimerState(record, time.Now())
	if err != nil {
		writeError(w, "failed to update active timer", http.StatusInternalServerError)
		return
	}
	writeJSON(w, TimerStatusResponse{Active: true, Timer: &state}, http.StatusOK)
}

func startTimerHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	var request StartTimerRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	if request.SpeedMultiplier == 0 {
		request.SpeedMultiplier = 1
	}
	if !validTimerSpeed(request.SpeedMultiplier) {
		writeError(w, "speedMultiplier is not allowed", http.StatusBadRequest)
		return
	}

	todayMinutes, err := loadGoalMinutesForDate(goal.ID, todayStringForUser(goal.UserID))
	if err != nil {
		writeError(w, "failed to load today's progress", http.StatusInternalServerError)
		return
	}
	remainingMinutes := goal.DailyTargetMinutes - todayMinutes
	if remainingMinutes <= 0 {
		writeError(w, "daily target is already completed", http.StatusConflict)
		return
	}

	location := userLocation(goal.UserID)
	localNow := time.Now().In(location)
	now := localNow.UTC()
	secondsUntilMidnight := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, 0, 0, 0, 0, location).Sub(localNow).Seconds()
	targetSeconds := min(remainingMinutes*60, max(1, int(math.Ceil(secondsUntilMidnight*request.SpeedMultiplier))))
	result, err := db.Exec(`
		INSERT INTO active_timers (
			user_id, goal_id, session_date, state, started_at, last_resumed_at,
			accumulated_seconds, target_seconds, speed_multiplier, updated_at
		)
		VALUES (?, ?, ?, 'running', ?, ?, 0, ?, ?, ?)
	`, goal.UserID, goal.ID, localNow.Format(time.DateOnly), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano),
		targetSeconds, request.SpeedMultiplier, now.Format(time.RFC3339Nano))
	if err != nil {
		writeError(w, "another timer is already active", http.StatusConflict)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, "failed to read active timer", http.StatusInternalServerError)
		return
	}
	record := activeTimerRecord{
		ID:              int(id),
		UserID:          goal.UserID,
		GoalID:          goal.ID,
		SessionDate:     localNow.Format(time.DateOnly),
		State:           "running",
		StartedAt:       now.Format(time.RFC3339Nano),
		LastResumedAt:   now.Format(time.RFC3339Nano),
		TargetSeconds:   targetSeconds,
		SpeedMultiplier: request.SpeedMultiplier,
	}
	state, _, err := timerState(record, now)
	if err != nil {
		writeError(w, "failed to initialize active timer", http.StatusInternalServerError)
		return
	}
	writeJSON(w, state, http.StatusCreated)
}

func pauseTimerHandler(w http.ResponseWriter, r *http.Request) {
	changeTimerStateHandler(w, r, "paused")
}

func resumeTimerHandler(w http.ResponseWriter, r *http.Request) {
	changeTimerStateHandler(w, r, "running")
}

func changeTimerStateHandler(w http.ResponseWriter, r *http.Request, targetState string) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}
	record, found, err := loadActiveTimer(goal.UserID)
	if err != nil {
		writeError(w, "failed to load active timer", http.StatusInternalServerError)
		return
	}
	if !found || record.GoalID != goal.ID {
		writeError(w, "active timer not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()
	state, elapsed, err := timerState(record, now)
	if err != nil {
		writeError(w, "failed to read active timer", http.StatusInternalServerError)
		return
	}
	if state.State == "finished" {
		if err := persistTimerClock(record.ID, "finished", float64(record.TargetSeconds), now); err != nil {
			writeError(w, "failed to stop active timer", http.StatusInternalServerError)
			return
		}
		writeJSON(w, state, http.StatusOK)
		return
	}

	if targetState == "running" && record.State != "paused" {
		writeError(w, "timer is not paused", http.StatusConflict)
		return
	}
	if targetState == "paused" && record.State != "running" {
		writeError(w, "timer is not running", http.StatusConflict)
		return
	}

	if err := persistTimerClock(record.ID, targetState, elapsed, now); err != nil {
		writeError(w, "failed to update active timer", http.StatusInternalServerError)
		return
	}
	record.State = targetState
	record.AccumulatedSeconds = elapsed
	record.LastResumedAt = now.Format(time.RFC3339Nano)
	updated, _, err := timerState(record, now)
	if err != nil {
		writeError(w, "failed to read updated timer", http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated, http.StatusOK)
}

func finishTimerHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}
	var request FinishTimerRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Notes = strings.TrimSpace(request.Notes)
	request.Tags = cleanTags(request.Tags)
	if message := validateSessionContent(request.Notes, request.Tags); message != "" {
		writeError(w, message, http.StatusBadRequest)
		return
	}

	record, found, err := loadActiveTimer(goal.UserID)
	if err != nil {
		writeError(w, "failed to load active timer", http.StatusInternalServerError)
		return
	}
	if !found || record.GoalID != goal.ID {
		writeError(w, "active timer not found", http.StatusNotFound)
		return
	}

	now := time.Now().UTC()
	_, elapsed, err := timerState(record, now)
	if err != nil {
		writeError(w, "failed to read active timer", http.StatusInternalServerError)
		return
	}
	durationMinutes := max(1, int(math.Ceil(elapsed/60)))
	session, status, err := saveFinishedTimer(goal, record, durationMinutes, request, now)
	if err != nil {
		writeError(w, "failed to finish active timer", http.StatusInternalServerError)
		return
	}
	if err := refreshDailyProgressForGoal(goal); err != nil {
		writeError(w, "failed to refresh daily progress", http.StatusInternalServerError)
		return
	}
	writeJSON(w, session, status)
}

func loadActiveTimer(userID int) (activeTimerRecord, bool, error) {
	var record activeTimerRecord
	err := db.QueryRow(`
		SELECT id, user_id, goal_id, session_date, state, started_at, last_resumed_at,
			accumulated_seconds, target_seconds, speed_multiplier
		FROM active_timers
		WHERE user_id = ?
	`, userID).Scan(
		&record.ID, &record.UserID, &record.GoalID, &record.SessionDate, &record.State,
		&record.StartedAt, &record.LastResumedAt, &record.AccumulatedSeconds,
		&record.TargetSeconds, &record.SpeedMultiplier,
	)
	if err == sql.ErrNoRows {
		return activeTimerRecord{}, false, nil
	}
	return record, err == nil, err
}

func currentTimerState(record activeTimerRecord, now time.Time) (TimerState, error) {
	state, elapsed, err := timerState(record, now)
	if err != nil {
		return TimerState{}, err
	}
	if state.State == "finished" && record.State != "finished" {
		if err := persistTimerClock(record.ID, "finished", elapsed, now); err != nil {
			return TimerState{}, err
		}
	}
	return state, nil
}

func timerState(record activeTimerRecord, now time.Time) (TimerState, float64, error) {
	elapsed := record.AccumulatedSeconds
	if record.State == "running" {
		lastResumed, err := time.Parse(time.RFC3339Nano, record.LastResumedAt)
		if err != nil {
			return TimerState{}, 0, err
		}
		if seconds := now.Sub(lastResumed).Seconds(); seconds > 0 {
			elapsed += seconds * record.SpeedMultiplier
		}
	}
	state := record.State
	if elapsed >= float64(record.TargetSeconds) {
		elapsed = float64(record.TargetSeconds)
		state = "finished"
	}
	return TimerState{
		GoalID:          record.GoalID,
		State:           state,
		StartedAt:       record.StartedAt,
		ElapsedSeconds:  int(math.Floor(elapsed)),
		TargetSeconds:   record.TargetSeconds,
		SpeedMultiplier: record.SpeedMultiplier,
	}, elapsed, nil
}

func persistTimerClock(id int, state string, elapsed float64, now time.Time) error {
	_, err := db.Exec(`
		UPDATE active_timers
		SET state = ?, accumulated_seconds = ?, last_resumed_at = ?, updated_at = ?
		WHERE id = ?
	`, state, elapsed, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id)
	return err
}

func saveFinishedTimer(goal Goal, timer activeTimerRecord, durationMinutes int, request FinishTimerRequest, now time.Time) (Session, int, error) {
	localEnd := now.In(time.Local)
	if localEnd.Format(time.DateOnly) != timer.SessionDate {
		startOfNextDay, err := time.ParseInLocation(time.DateOnly, timer.SessionDate, time.Local)
		if err != nil {
			return Session{}, 0, err
		}
		localEnd = startOfNextDay.AddDate(0, 0, 1).Add(-time.Second)
	}
	endedAt := localEnd.Format(time.RFC3339)
	sessionDate := timer.SessionDate
	createdAt := now.Format(time.RFC3339)
	tx, err := db.Begin()
	if err != nil {
		return Session{}, 0, err
	}
	defer tx.Rollback()

	existing, found, err := loadSessionForDateWith(tx, goal.ID, sessionDate)
	if err != nil {
		return Session{}, 0, err
	}
	status := http.StatusCreated
	var session Session
	if found {
		durationMinutes = min(existing.DurationMinutes+durationMinutes, maxDailyMinutes)
		notes := mergeSessionNotes(existing.Notes, request.Notes)
		tags := mergeTags(existing.Tags, request.Tags)
		if _, err := tx.Exec(`
			UPDATE sessions
			SET ended_at = ?, duration_minutes = ?, notes = ?, tags = ?
			WHERE id = ? AND goal_id = ?
		`, endedAt, durationMinutes, notes, tagsToString(tags), existing.ID, goal.ID); err != nil {
			return Session{}, 0, err
		}
		session, err = loadSessionWith(tx, goal.ID, existing.ID)
		status = http.StatusOK
	} else {
		result, insertErr := tx.Exec(`
			INSERT INTO sessions (
				goal_id, started_at, ended_at, duration_minutes,
				notes, tags, created_at, session_date
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, goal.ID, timer.StartedAt, endedAt, durationMinutes, request.Notes,
			tagsToString(request.Tags), createdAt, sessionDate)
		if insertErr != nil {
			return Session{}, 0, insertErr
		}
		id, idErr := result.LastInsertId()
		if idErr != nil {
			return Session{}, 0, idErr
		}
		session, err = loadSessionWith(tx, goal.ID, int(id))
	}
	if err != nil {
		return Session{}, 0, err
	}
	if _, err := tx.Exec(`DELETE FROM active_timers WHERE id = ?`, timer.ID); err != nil {
		return Session{}, 0, err
	}
	if err := tx.Commit(); err != nil {
		return Session{}, 0, err
	}
	return session, status, nil
}

func validTimerSpeed(speed float64) bool {
	if !developmentTimerSpeedEnabled() {
		return speed == 1
	}
	for _, allowed := range []float64{0.5, 1, 1.5, 2, 5} {
		if speed == allowed {
			return true
		}
	}
	return false
}
