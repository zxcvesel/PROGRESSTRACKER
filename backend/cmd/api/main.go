package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

type Entry struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Category string `json:"category"`
	Minutes  int    `json:"minutes"`
	Note     string `json:"note"`
}

type Goal struct {
	ID                 int    `json:"id"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	TotalDays          int    `json:"totalDays"`
	DailyTargetMinutes int    `json:"dailyTargetMinutes"`
	ActiveWeekdays     []int  `json:"activeWeekdays"`
	StartDate          string `json:"startDate"`
	CreatedAt          string `json:"createdAt"`
	Status             string `json:"status"`
}

type GoalSummary struct {
	Goal
	CurrentStreak       int `json:"currentStreak"`
	TodayMinutes        int `json:"todayMinutes"`
	TodayProgressPct    int `json:"todayProgressPct"`
	CurrentDay          int `json:"currentDay"`
	TotalProgressPct    int `json:"totalProgressPct"`
	TotalPracticeMinute int `json:"totalPracticeMinutes"`
}

type Session struct {
	ID              int      `json:"id"`
	GoalID          int      `json:"goalId"`
	StartedAt       string   `json:"startedAt"`
	EndedAt         string   `json:"endedAt"`
	DurationMinutes int      `json:"durationMinutes"`
	Notes           string   `json:"notes"`
	Tags            []string `json:"tags"`
	CreatedAt       string   `json:"createdAt"`
}

type GoalDetail struct {
	GoalSummary
	TodayRemainingMinutes int       `json:"todayRemainingMinutes"`
	RecentSessions        []Session `json:"recentSessions"`
}

type CreateGoalRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	TotalDays          int    `json:"totalDays"`
	DailyTargetMinutes int    `json:"dailyTargetMinutes"`
	ActiveWeekdays     []int  `json:"activeWeekdays"`
	StartDate          string `json:"startDate"`
}

type UpdateGoalRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	TotalDays          int    `json:"totalDays"`
	DailyTargetMinutes int    `json:"dailyTargetMinutes"`
	Status             string `json:"status"`
}

type CreateSessionRequest struct {
	StartedAt       string   `json:"startedAt"`
	EndedAt         string   `json:"endedAt"`
	DurationMinutes int      `json:"durationMinutes"`
	Notes           string   `json:"notes"`
	Tags            []string `json:"tags"`
}

type UpdateSessionRequest struct {
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`
}

type Stats struct {
	TotalSessions        int                `json:"totalSessions"`
	TotalPracticeMinutes int                `json:"totalPracticeMinutes"`
	CurrentStreak        int                `json:"currentStreak"`
	LongestStreak        int                `json:"longestStreak"`
	TodayMinutes         int                `json:"todayMinutes"`
	DailyTargetMinutes   int                `json:"dailyTargetMinutes"`
	Weekly               []DailyStat        `json:"weekly"`
	MonthlyTotalMinutes  int                `json:"monthlyTotalMinutes"`
	GoalDistribution     []GoalDistribution `json:"goalDistribution"`
}

type DailyStat struct {
	Date          string `json:"date"`
	Label         string `json:"label"`
	Minutes       int    `json:"minutes"`
	TargetMinutes int    `json:"targetMinutes"`
	IsCompleted   bool   `json:"isCompleted"`
}

type GoalDistribution struct {
	GoalID  int    `json:"goalId"`
	Title   string `json:"title"`
	Minutes int    `json:"minutes"`
	Percent int    `json:"percent"`
}

func main() {
	database, err := openDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	db = database

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /entries", entriesHandler)
	mux.HandleFunc("POST /entries", createEntryHandler)
	mux.HandleFunc("GET /goals", goalsHandler)
	mux.HandleFunc("POST /goals", createGoalHandler)
	mux.HandleFunc("GET /goals/{id}", goalDetailHandler)
	mux.HandleFunc("PATCH /goals/{id}", updateGoalHandler)
	mux.HandleFunc("DELETE /goals/{id}", deleteGoalHandler)
	mux.HandleFunc("POST /goals/{id}/sessions", createSessionHandler)
	mux.HandleFunc("PATCH /goals/{id}/sessions/{sessionId}", updateSessionHandler)
	mux.HandleFunc("DELETE /goals/{id}/sessions/{sessionId}", deleteSessionHandler)
	mux.HandleFunc("GET /stats", statsHandler)

	port := os.Getenv("PROGRESS_TRACKER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Backend is running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func openDatabase() (*sql.DB, error) {
	dbPath := os.Getenv("PROGRESS_TRACKER_DB_PATH")
	if dbPath == "" {
		dbPath = "data/progress.db"
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	queries := []string{
		`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT NOT NULL,
			category TEXT NOT NULL,
			minutes INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT ''
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			total_days INTEGER NOT NULL,
			daily_target_minutes INTEGER NOT NULL,
			active_weekdays TEXT NOT NULL,
			start_date TEXT NOT NULL,
			created_at TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active'
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			goal_id INTEGER NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			notes TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(goal_id) REFERENCES goals(id)
		);
		`,
	}

	for _, query := range queries {
		if _, err := database.Exec(query); err != nil {
			database.Close()
			return nil, err
		}
	}

	return database, nil
}

func entriesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, date, category, minutes, note
		FROM entries
		ORDER BY date DESC, id DESC
	`)
	if err != nil {
		http.Error(w, "failed to load entries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Date, &entry.Category, &entry.Minutes, &entry.Note); err != nil {
			http.Error(w, "failed to read entry", http.StatusInternalServerError)
			return
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to load entries", http.StatusInternalServerError)
		return
	}

	writeJSON(w, entries, http.StatusOK)
}

func createEntryHandler(w http.ResponseWriter, r *http.Request) {
	var entry Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if entry.Date == "" || entry.Category == "" || entry.Minutes <= 0 {
		http.Error(w, "date, category, and positive minutes are required", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		INSERT INTO entries (date, category, minutes, note)
		VALUES (?, ?, ?, ?)
	`, entry.Date, entry.Category, entry.Minutes, entry.Note)
	if err != nil {
		http.Error(w, "failed to save entry", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "failed to read created entry", http.StatusInternalServerError)
		return
	}

	entry.ID = int(id)
	writeJSON(w, entry, http.StatusCreated)
}

func goalsHandler(w http.ResponseWriter, r *http.Request) {
	goals, err := loadGoals()
	if err != nil {
		http.Error(w, "failed to load goals", http.StatusInternalServerError)
		return
	}

	summaries := make([]GoalSummary, 0, len(goals))
	for _, goal := range goals {
		summary, err := buildGoalSummary(goal)
		if err != nil {
			http.Error(w, "failed to load goal summary", http.StatusInternalServerError)
			return
		}
		summaries = append(summaries, summary)
	}

	writeJSON(w, summaries, http.StatusOK)
}

func createGoalHandler(w http.ResponseWriter, r *http.Request) {
	var request CreateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Title == "" || request.TotalDays <= 0 || request.DailyTargetMinutes <= 0 {
		http.Error(w, "title, totalDays, and dailyTargetMinutes are required", http.StatusBadRequest)
		return
	}

	if len(request.ActiveWeekdays) == 0 {
		request.ActiveWeekdays = []int{1, 2, 3, 4, 5, 6, 7}
	}

	if request.StartDate == "" {
		request.StartDate = todayString()
	}

	createdAt := time.Now().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO goals (
			title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
	`, request.Title, request.Description, request.TotalDays, request.DailyTargetMinutes, weekdaysToString(request.ActiveWeekdays), request.StartDate, createdAt)
	if err != nil {
		http.Error(w, "failed to create goal", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "failed to read created goal", http.StatusInternalServerError)
		return
	}

	goal := Goal{
		ID:                 int(id),
		Title:              request.Title,
		Description:        request.Description,
		TotalDays:          request.TotalDays,
		DailyTargetMinutes: request.DailyTargetMinutes,
		ActiveWeekdays:     request.ActiveWeekdays,
		StartDate:          request.StartDate,
		CreatedAt:          createdAt,
		Status:             "active",
	}

	summary, err := buildGoalSummary(goal)
	if err != nil {
		http.Error(w, "failed to load created goal", http.StatusInternalServerError)
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
		http.Error(w, "failed to load goal summary", http.StatusInternalServerError)
		return
	}

	sessions, err := loadSessions(goal.ID, 12)
	if err != nil {
		http.Error(w, "failed to load sessions", http.StatusInternalServerError)
		return
	}

	detail := GoalDetail{
		GoalSummary:           summary,
		TodayRemainingMinutes: max(goal.DailyTargetMinutes-summary.TodayMinutes, 0),
		RecentSessions:        sessions,
	}

	writeJSON(w, detail, http.StatusOK)
}

func updateGoalHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	var request UpdateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Title == "" || request.TotalDays <= 0 || request.DailyTargetMinutes <= 0 {
		http.Error(w, "title, totalDays, and dailyTargetMinutes are required", http.StatusBadRequest)
		return
	}

	if request.Status == "" {
		request.Status = goal.Status
	}
	if request.Status != "active" && request.Status != "completed" {
		http.Error(w, "status must be active or completed", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		UPDATE goals
		SET title = ?, description = ?, total_days = ?, daily_target_minutes = ?, status = ?
		WHERE id = ?
	`, request.Title, request.Description, request.TotalDays, request.DailyTargetMinutes, request.Status, goal.ID)
	if err != nil {
		http.Error(w, "failed to update goal", http.StatusInternalServerError)
		return
	}

	updatedGoal, err := loadGoal(goal.ID)
	if err != nil {
		http.Error(w, "failed to load updated goal", http.StatusInternalServerError)
		return
	}

	summary, err := buildGoalSummary(updatedGoal)
	if err != nil {
		http.Error(w, "failed to load updated goal summary", http.StatusInternalServerError)
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
		http.Error(w, "failed to start delete", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions WHERE goal_id = ?`, goal.ID); err != nil {
		http.Error(w, "failed to delete goal sessions", http.StatusInternalServerError)
		return
	}

	if _, err := tx.Exec(`DELETE FROM goals WHERE id = ?`, goal.ID); err != nil {
		http.Error(w, "failed to delete goal", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to finish delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createSessionHandler(w http.ResponseWriter, r *http.Request) {
	goal, ok := loadGoalFromRequest(w, r)
	if !ok {
		return
	}

	var request CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if request.StartedAt == "" || request.EndedAt == "" || request.DurationMinutes <= 0 {
		http.Error(w, "startedAt, endedAt, and positive durationMinutes are required", http.StatusBadRequest)
		return
	}

	sessionDate := sessionDateString(request.EndedAt)
	createdAt := time.Now().Format(time.RFC3339)
	existingSession, hasExistingSession, err := loadSessionForDate(goal.ID, sessionDate)
	if err != nil {
		http.Error(w, "failed to load daily session", http.StatusInternalServerError)
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
			http.Error(w, "failed to update daily session", http.StatusInternalServerError)
			return
		}

		session, err := loadSession(goal.ID, existingSession.ID)
		if err != nil {
			http.Error(w, "failed to load daily session", http.StatusInternalServerError)
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
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, "failed to read created session", http.StatusInternalServerError)
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
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		UPDATE sessions
		SET notes = ?, tags = ?
		WHERE id = ? AND goal_id = ?
	`, request.Notes, tagsToString(request.Tags), sessionID, goal.ID)
	if err != nil {
		http.Error(w, "failed to update session", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "failed to read updated session", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	session, err := loadSession(goal.ID, sessionID)
	if err != nil {
		http.Error(w, "failed to load updated session", http.StatusInternalServerError)
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
		http.Error(w, "failed to delete session", http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "failed to read deleted session", http.StatusInternalServerError)
		return
	}
	if affected == 0 {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	goals, err := loadGoals()
	if err != nil {
		http.Error(w, "failed to load goals", http.StatusInternalServerError)
		return
	}

	totalSessions, totalMinutes, err := loadSessionTotals()
	if err != nil {
		http.Error(w, "failed to load stats", http.StatusInternalServerError)
		return
	}

	today := todayString()
	todayMinutes, err := loadMinutesForDate(today)
	if err != nil {
		http.Error(w, "failed to load today's stats", http.StatusInternalServerError)
		return
	}

	dailyTarget := 0
	currentStreak := 0
	longestStreak := 0
	for _, goal := range goals {
		if goal.Status == "active" {
			dailyTarget += goal.DailyTargetMinutes
		}

		current, longest, err := calculateGoalStreaks(goal)
		if err != nil {
			http.Error(w, "failed to load streaks", http.StatusInternalServerError)
			return
		}

		currentStreak = max(currentStreak, current)
		longestStreak = max(longestStreak, longest)
	}

	weekly, err := buildWeeklyStats(dailyTarget)
	if err != nil {
		http.Error(w, "failed to load weekly stats", http.StatusInternalServerError)
		return
	}

	monthlyTotal, err := loadMonthlyTotal()
	if err != nil {
		http.Error(w, "failed to load monthly stats", http.StatusInternalServerError)
		return
	}

	distribution, err := loadGoalDistribution(totalMinutes)
	if err != nil {
		http.Error(w, "failed to load distribution", http.StatusInternalServerError)
		return
	}

	stats := Stats{
		TotalSessions:        totalSessions,
		TotalPracticeMinutes: totalMinutes,
		CurrentStreak:        currentStreak,
		LongestStreak:        longestStreak,
		TodayMinutes:         todayMinutes,
		DailyTargetMinutes:   dailyTarget,
		Weekly:               weekly,
		MonthlyTotalMinutes:  monthlyTotal,
		GoalDistribution:     distribution,
	}

	writeJSON(w, stats, http.StatusOK)
}

func loadGoalFromRequest(w http.ResponseWriter, r *http.Request) (Goal, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid goal id", http.StatusBadRequest)
		return Goal{}, false
	}

	goal, err := loadGoal(id)
	if err == sql.ErrNoRows {
		http.Error(w, "goal not found", http.StatusNotFound)
		return Goal{}, false
	}
	if err != nil {
		http.Error(w, "failed to load goal", http.StatusInternalServerError)
		return Goal{}, false
	}

	return goal, true
}

func sessionIDFromRequest(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("sessionId"))
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return 0, false
	}

	return id, true
}

func loadGoals() ([]Goal, error) {
	rows, err := db.Query(`
		SELECT id, title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status
		FROM goals
		ORDER BY status ASC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	goals := []Goal{}
	for rows.Next() {
		goal, err := scanGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}

	return goals, rows.Err()
}

func loadGoal(id int) (Goal, error) {
	row := db.QueryRow(`
		SELECT id, title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status
		FROM goals
		WHERE id = ?
	`, id)

	return scanGoal(row)
}

type goalScanner interface {
	Scan(dest ...any) error
}

func scanGoal(scanner goalScanner) (Goal, error) {
	var goal Goal
	var activeWeekdays string

	err := scanner.Scan(
		&goal.ID,
		&goal.Title,
		&goal.Description,
		&goal.TotalDays,
		&goal.DailyTargetMinutes,
		&activeWeekdays,
		&goal.StartDate,
		&goal.CreatedAt,
		&goal.Status,
	)
	if err != nil {
		return Goal{}, err
	}

	goal.ActiveWeekdays = parseActiveWeekdays(activeWeekdays)
	return goal, nil
}

func buildGoalSummary(goal Goal) (GoalSummary, error) {
	todayMinutes, err := loadGoalMinutesForDate(goal.ID, todayString())
	if err != nil {
		return GoalSummary{}, err
	}

	totalMinutes, err := loadGoalTotalMinutes(goal.ID)
	if err != nil {
		return GoalSummary{}, err
	}

	currentStreak, _, completedDaysCount, err := calculateGoalProgress(goal)
	if err != nil {
		return GoalSummary{}, err
	}

	currentDay := min(completedDaysCount, goal.TotalDays)
	return GoalSummary{
		Goal:                goal,
		CurrentStreak:       currentStreak,
		TodayMinutes:        todayMinutes,
		TodayProgressPct:    percent(todayMinutes, goal.DailyTargetMinutes),
		CurrentDay:          currentDay,
		TotalProgressPct:    percent(currentDay, goal.TotalDays),
		TotalPracticeMinute: totalMinutes,
	}, nil
}

func loadSessions(goalID int, limit int) ([]Session, error) {
	rows, err := db.Query(`
		SELECT id, goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at
		FROM sessions
		WHERE goal_id = ?
		ORDER BY ended_at DESC, id DESC
		LIMIT ?
	`, goalID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []Session{}
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

func loadSession(goalID int, sessionID int) (Session, error) {
	row := db.QueryRow(`
		SELECT id, goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at
		FROM sessions
		WHERE goal_id = ? AND id = ?
	`, goalID, sessionID)

	return scanSession(row)
}

func loadSessionForDate(goalID int, date string) (Session, bool, error) {
	rows, err := db.Query(`
		SELECT id, goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at
		FROM sessions
		WHERE goal_id = ?
		ORDER BY ended_at DESC, id DESC
	`, goalID)
	if err != nil {
		return Session{}, false, err
	}
	defer rows.Close()

	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return Session{}, false, err
		}
		if sessionDateString(session.EndedAt) == date {
			return session, true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return Session{}, false, err
	}

	return Session{}, false, nil
}

type sessionScanner interface {
	Scan(dest ...any) error
}

func scanSession(scanner sessionScanner) (Session, error) {
	var session Session
	var tags string

	if err := scanner.Scan(
		&session.ID,
		&session.GoalID,
		&session.StartedAt,
		&session.EndedAt,
		&session.DurationMinutes,
		&session.Notes,
		&tags,
		&session.CreatedAt,
	); err != nil {
		return Session{}, err
	}

	session.Tags = parseTags(tags)
	return session, nil
}

func loadGoalMinutesForDate(goalID int, date string) (int, error) {
	rows, err := db.Query(`
		SELECT ended_at, duration_minutes
		FROM sessions
		WHERE goal_id = ?
	`, goalID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	minutes := 0
	for rows.Next() {
		var endedAt string
		var durationMinutes int
		if err := rows.Scan(&endedAt, &durationMinutes); err != nil {
			return 0, err
		}
		if sessionDateString(endedAt) == date {
			minutes += durationMinutes
		}
	}

	return minutes, rows.Err()
}

func loadMinutesForDate(date string) (int, error) {
	rows, err := db.Query(`
		SELECT ended_at, duration_minutes
		FROM sessions
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	minutes := 0
	for rows.Next() {
		var endedAt string
		var durationMinutes int
		if err := rows.Scan(&endedAt, &durationMinutes); err != nil {
			return 0, err
		}
		if sessionDateString(endedAt) == date {
			minutes += durationMinutes
		}
	}

	return minutes, rows.Err()
}

func loadGoalTotalMinutes(goalID int) (int, error) {
	var minutes int
	err := db.QueryRow(`
		SELECT COALESCE(SUM(duration_minutes), 0)
		FROM sessions
		WHERE goal_id = ?
	`, goalID).Scan(&minutes)
	return minutes, err
}

func loadSessionTotals() (int, int, error) {
	var sessions int
	var minutes int
	err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(duration_minutes), 0)
		FROM sessions
	`).Scan(&sessions, &minutes)
	return sessions, minutes, err
}

func calculateGoalStreaks(goal Goal) (int, int, error) {
	current, longest, _, err := calculateGoalProgress(goal)
	return current, longest, err
}

func calculateGoalProgress(goal Goal) (int, int, int, error) {
	rows, err := db.Query(`
		SELECT ended_at, duration_minutes
		FROM sessions
		WHERE goal_id = ?
	`, goal.ID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	dayMinutes := map[string]int{}
	for rows.Next() {
		var endedAt string
		var durationMinutes int
		if err := rows.Scan(&endedAt, &durationMinutes); err != nil {
			return 0, 0, 0, err
		}

		dayMinutes[sessionDateString(endedAt)] += durationMinutes
	}

	if err := rows.Err(); err != nil {
		return 0, 0, 0, err
	}

	completedDays := map[string]bool{}
	completedDaysCount := 0
	for date, minutes := range dayMinutes {
		isCompleted := minutes >= goal.DailyTargetMinutes
		completedDays[date] = isCompleted
		if isCompleted {
			completedDaysCount++
		}
	}

	current := 0
	cursor := time.Now()
	for {
		date := cursor.Format(time.DateOnly)
		if !completedDays[date] {
			break
		}
		current++
		cursor = cursor.AddDate(0, 0, -1)
	}

	longest := 0
	running := 0
	for day := 0; day < goal.TotalDays; day++ {
		date := parseDateOrToday(goal.StartDate).AddDate(0, 0, day).Format(time.DateOnly)
		if completedDays[date] {
			running++
			longest = max(longest, running)
			continue
		}
		running = 0
	}

	return current, longest, completedDaysCount, nil
}

func buildWeeklyStats(dailyTarget int) ([]DailyStat, error) {
	stats := make([]DailyStat, 0, 7)
	today := time.Now()
	start := today.AddDate(0, 0, -6)

	for day := 0; day < 7; day++ {
		date := start.AddDate(0, 0, day)
		dateString := date.Format(time.DateOnly)
		minutes, err := loadMinutesForDate(dateString)
		if err != nil {
			return nil, err
		}

		stats = append(stats, DailyStat{
			Date:          dateString,
			Label:         weekdayLabel(date),
			Minutes:       minutes,
			TargetMinutes: dailyTarget,
			IsCompleted:   dailyTarget > 0 && minutes >= dailyTarget,
		})
	}

	return stats, nil
}

func loadMonthlyTotal() (int, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(time.DateOnly)
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Format(time.DateOnly)

	rows, err := db.Query(`
		SELECT ended_at, duration_minutes
		FROM sessions
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	minutes := 0
	for rows.Next() {
		var endedAt string
		var durationMinutes int
		if err := rows.Scan(&endedAt, &durationMinutes); err != nil {
			return 0, err
		}

		date := sessionDateString(endedAt)
		if date >= monthStart && date < nextMonth {
			minutes += durationMinutes
		}
	}

	return minutes, rows.Err()
}

func loadGoalDistribution(totalMinutes int) ([]GoalDistribution, error) {
	rows, err := db.Query(`
		SELECT goals.id, goals.title, COALESCE(SUM(sessions.duration_minutes), 0)
		FROM goals
		LEFT JOIN sessions ON sessions.goal_id = goals.id
		GROUP BY goals.id, goals.title
		ORDER BY COALESCE(SUM(sessions.duration_minutes), 0) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distribution := []GoalDistribution{}
	for rows.Next() {
		var item GoalDistribution
		if err := rows.Scan(&item.GoalID, &item.Title, &item.Minutes); err != nil {
			return nil, err
		}
		item.Percent = percent(item.Minutes, totalMinutes)
		distribution = append(distribution, item)
	}

	return distribution, rows.Err()
}

func writeJSON(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func weekdaysToString(weekdays []int) string {
	values := make([]string, 0, len(weekdays))
	for _, weekday := range weekdays {
		if weekday >= 1 && weekday <= 7 {
			values = append(values, strconv.Itoa(weekday))
		}
	}
	return strings.Join(values, ",")
}

func parseActiveWeekdays(value string) []int {
	if value == "" {
		return []int{}
	}

	parts := strings.Split(value, ",")
	weekdays := make([]int, 0, len(parts))
	for _, part := range parts {
		weekday, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			weekdays = append(weekdays, weekday)
		}
	}
	return weekdays
}

func tagsToString(tags []string) string {
	return strings.Join(cleanTags(tags), ",")
}

func parseTags(value string) []string {
	if value == "" {
		return []string{}
	}
	return cleanTags(strings.Split(value, ","))
}

func cleanTags(tags []string) []string {
	result := []string{}
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func mergeSessionNotes(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)

	if current == "" {
		return next
	}
	if next == "" || current == next {
		return current
	}

	return current + "\n" + next
}

func mergeTags(current []string, next []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, tag := range append(cleanTags(current), cleanTags(next)...) {
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, tag)
	}

	return result
}

func todayString() string {
	return time.Now().Format(time.DateOnly)
}

func sessionDateString(value string) string {
	if timestamp, err := time.Parse(time.RFC3339Nano, value); err == nil {
		if strings.HasSuffix(value, "Z") {
			return timestamp.Local().Format(time.DateOnly)
		}
		return timestamp.Format(time.DateOnly)
	}

	if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
		if strings.HasSuffix(value, "Z") {
			return timestamp.Local().Format(time.DateOnly)
		}
		return timestamp.Format(time.DateOnly)
	}

	if timestamp, err := time.ParseInLocation(time.DateOnly, value, time.Local); err == nil {
		return timestamp.Format(time.DateOnly)
	}

	if len(value) >= len(time.DateOnly) {
		return value[:len(time.DateOnly)]
	}

	return todayString()
}

func parseDateOrToday(value string) time.Time {
	date, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Now()
	}
	return date
}

func getGoalDay(startDate string, durationDays int) int {
	start := parseDateOrToday(startDate)
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	diff := today.Sub(start)
	day := int(math.Floor(diff.Hours()/24)) + 1
	return min(max(day, 1), durationDays)
}

func percent(value int, total int) int {
	if total <= 0 {
		return 0
	}
	return min(int(math.Round((float64(value)/float64(total))*100)), 100)
}

func weekdayLabel(date time.Time) string {
	labels := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	return labels[int(date.Weekday())]
}

func min(first int, second int) int {
	if first < second {
		return first
	}
	return second
}

func max(first int, second int) int {
	if first > second {
		return first
	}
	return second
}
