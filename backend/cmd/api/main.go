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

const (
	authTokenLifetime = 30 * 24 * time.Hour
	authCookieName    = "progress_tracker_session"
	passwordHashName  = "argon2id"
	passwordKeyBytes  = 32
	passwordSaltBytes = 16
	passwordMemory    = 64 * 1024
	passwordTime      = 3
	passwordThreads   = 1
	legacyHashName    = "pbkdf2_sha256"
	legacyRounds      = 120000
)

var authRateLimiter = newRateLimiter(12, 10*time.Minute)

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"createdAt"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type AuthResponse struct {
	User User `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Entry struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Category string `json:"category"`
	Minutes  int    `json:"minutes"`
	Note     string `json:"note"`
}

type Goal struct {
	ID                 int    `json:"id"`
	UserID             int    `json:"-"`
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
	TodayRemainingMinutes int         `json:"todayRemainingMinutes"`
	RecentSessions        []Session   `json:"recentSessions"`
	Calendar              []DailyStat `json:"calendar"`
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
	CompletedDays        int                `json:"completedDays"`
	MissedDays           int                `json:"missedDays"`
	CompletionRate       int                `json:"completionRate"`
	WeeklyCompletionRate int                `json:"weeklyCompletionRate"`
	PreviousWeekMinutes  int                `json:"previousWeekMinutes"`
	WeekComparisonPct    int                `json:"weekComparisonPct"`
	TodayMinutes         int                `json:"todayMinutes"`
	DailyTargetMinutes   int                `json:"dailyTargetMinutes"`
	Weekly               []DailyStat        `json:"weekly"`
	Calendar             []DailyStat        `json:"calendar"`
	MonthlyTotalMinutes  int                `json:"monthlyTotalMinutes"`
	GoalDistribution     []GoalDistribution `json:"goalDistribution"`
	GoalID               int                `json:"goalId"`
	GoalTitle            string             `json:"goalTitle"`
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
	if err := refreshDailyProgressForAllGoals(); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /auth/register", registerHandler)
	mux.HandleFunc("POST /auth/login", loginHandler)
	mux.HandleFunc("POST /auth/logout", logoutHandler)
	mux.HandleFunc("GET /me", meHandler)
	mux.HandleFunc("PATCH /me", updateMeHandler)
	mux.HandleFunc("PATCH /me/password", changePasswordHandler)
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
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS auth_sessions (
			token_hash TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);
		`,
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
		`
		CREATE TABLE IF NOT EXISTS daily_progress (
			goal_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			total_minutes INTEGER NOT NULL DEFAULT 0,
			target_minutes INTEGER NOT NULL,
			is_completed INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(goal_id, date),
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

	if err := addColumnIfMissing(database, "goals", "user_id", "user_id INTEGER NOT NULL DEFAULT 1"); err != nil {
		database.Close()
		return nil, err
	}
	if err := addColumnIfMissing(database, "entries", "user_id", "user_id INTEGER NOT NULL DEFAULT 1"); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func entriesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	rows, err := db.Query(`
		SELECT id, date, category, minutes, note
		FROM entries
		WHERE user_id = ?
		ORDER BY date DESC, id DESC
	`, user.ID)
	if err != nil {
		writeError(w, "failed to load entries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Date, &entry.Category, &entry.Minutes, &entry.Note); err != nil {
			writeError(w, "failed to read entry", http.StatusInternalServerError)
			return
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		writeError(w, "failed to load entries", http.StatusInternalServerError)
		return
	}

	writeJSON(w, entries, http.StatusOK)
}

func createEntryHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	var entry Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if entry.Date == "" || entry.Category == "" || entry.Minutes <= 0 {
		writeError(w, "date, category, and positive minutes are required", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		INSERT INTO entries (date, category, minutes, note, user_id)
		VALUES (?, ?, ?, ?, ?)
	`, entry.Date, entry.Category, entry.Minutes, entry.Note, user.ID)
	if err != nil {
		writeError(w, "failed to save entry", http.StatusInternalServerError)
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		writeError(w, "failed to read created entry", http.StatusInternalServerError)
		return
	}

	entry.ID = int(id)
	writeJSON(w, entry, http.StatusCreated)
}

func loadGoalFromRequest(w http.ResponseWriter, r *http.Request) (Goal, bool) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return Goal{}, false
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, "invalid goal id", http.StatusBadRequest)
		return Goal{}, false
	}

	goal, err := loadGoal(id, user.ID)
	if err == sql.ErrNoRows {
		writeError(w, "goal not found", http.StatusNotFound)
		return Goal{}, false
	}
	if err != nil {
		writeError(w, "failed to load goal", http.StatusInternalServerError)
		return Goal{}, false
	}

	return goal, true
}

func sessionIDFromRequest(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("sessionId"))
	if err != nil {
		writeError(w, "invalid session id", http.StatusBadRequest)
		return 0, false
	}

	return id, true
}

func optionalGoalID(r *http.Request) (int, error) {
	value := r.URL.Query().Get("goalId")
	if value == "" {
		return 0, nil
	}

	goalID, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if goalID <= 0 {
		return 0, strconv.ErrSyntax
	}

	return goalID, nil
}

func loadGoals(userID int) ([]Goal, error) {
	rows, err := db.Query(`
		SELECT id, user_id, title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status
		FROM goals
		WHERE user_id = ?
		ORDER BY status ASC, id DESC
	`, userID)
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

func loadAllGoals() ([]Goal, error) {
	rows, err := db.Query(`
		SELECT id, user_id, title, description, total_days, daily_target_minutes,
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

func loadGoal(id int, userID int) (Goal, error) {
	row := db.QueryRow(`
		SELECT id, user_id, title, description, total_days, daily_target_minutes,
			active_weekdays, start_date, created_at, status
		FROM goals
		WHERE id = ? AND user_id = ?
	`, id, userID)

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
		&goal.UserID,
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

func refreshDailyProgressForAllGoals() error {
	goals, err := loadAllGoals()
	if err != nil {
		return err
	}

	for _, goal := range goals {
		if err := refreshDailyProgressForGoal(goal); err != nil {
			return err
		}
	}

	return nil
}

func refreshDailyProgressForGoal(goal Goal) error {
	rows, err := db.Query(`
		SELECT ended_at, duration_minutes
		FROM sessions
		WHERE goal_id = ?
	`, goal.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	dayMinutes := map[string]int{}
	for rows.Next() {
		var endedAt string
		var durationMinutes int
		if err := rows.Scan(&endedAt, &durationMinutes); err != nil {
			return err
		}
		dayMinutes[sessionDateString(endedAt)] += durationMinutes
	}
	if err := rows.Err(); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM daily_progress WHERE goal_id = ?`, goal.ID); err != nil {
		return err
	}

	start := parseDateOnlyOrToday(goal.StartDate)
	today := dateOnly(time.Now())
	end := start.AddDate(0, 0, max(goal.TotalDays-1, 0))
	if end.After(today) {
		end = today
	}
	if start.After(end) {
		return tx.Commit()
	}

	statement, err := tx.Prepare(`
		INSERT INTO daily_progress (
			goal_id, date, total_minutes, target_minutes, is_completed
		)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer statement.Close()

	for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 0, 1) {
		date := cursor.Format(time.DateOnly)
		minutes := dayMinutes[date]
		isCompleted := 0
		if goal.DailyTargetMinutes > 0 && minutes >= goal.DailyTargetMinutes {
			isCompleted = 1
		}
		if _, err := statement.Exec(goal.ID, date, minutes, goal.DailyTargetMinutes, isCompleted); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func loadGoalMinutesForDate(goalID int, date string) (int, error) {
	var minutes int
	err := db.QueryRow(`
		SELECT COALESCE(total_minutes, 0)
		FROM daily_progress
		WHERE goal_id = ? AND date = ?
	`, goalID, date).Scan(&minutes)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return minutes, err
}

func loadMinutesForDate(date string, goalID int, userID int) (int, error) {
	var minutes int
	query := `
		SELECT COALESCE(SUM(total_minutes), 0)
		FROM daily_progress
		INNER JOIN goals ON goals.id = daily_progress.goal_id
		WHERE date = ? AND goals.user_id = ?
	`
	args := []any{date, userID}
	if goalID != 0 {
		query += ` AND goal_id = ?`
		args = append(args, goalID)
	}

	err := db.QueryRow(query, args...).Scan(&minutes)
	return minutes, err
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

func loadSessionTotals(goalID int, userID int) (int, int, error) {
	var sessions int
	var minutes int
	query := `
		SELECT COUNT(*), COALESCE(SUM(duration_minutes), 0)
		FROM sessions
		INNER JOIN goals ON goals.id = sessions.goal_id
		WHERE goals.user_id = ?
	`
	args := []any{userID}
	if goalID != 0 {
		query += ` AND goal_id = ?`
		args = append(args, goalID)
	}

	err := db.QueryRow(query, args...).Scan(&sessions, &minutes)
	return sessions, minutes, err
}

func calculateGoalStreaks(goal Goal) (int, int, error) {
	current, longest, _, err := calculateGoalProgress(goal)
	return current, longest, err
}

func calculateGoalProgress(goal Goal) (int, int, int, error) {
	rows, err := db.Query(`
		SELECT date, is_completed
		FROM daily_progress
		WHERE goal_id = ?
	`, goal.ID)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	completedDays := map[string]bool{}
	completedDaysCount := 0
	for rows.Next() {
		var date string
		var isCompleted int
		if err := rows.Scan(&date, &isCompleted); err != nil {
			return 0, 0, 0, err
		}

		completed := isCompleted == 1
		completedDays[date] = completed
		if completed {
			completedDaysCount++
		}
	}

	if err := rows.Err(); err != nil {
		return 0, 0, 0, err
	}

	current := 0
	cursor := time.Now()
	if !completedDays[cursor.Format(time.DateOnly)] {
		cursor = cursor.AddDate(0, 0, -1)
	}
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

func buildWeeklyStats(dailyTarget int, goalID int, userID int) ([]DailyStat, error) {
	stats := make([]DailyStat, 0, 7)
	today := time.Now()
	start := today.AddDate(0, 0, -6)

	for day := 0; day < 7; day++ {
		date := start.AddDate(0, 0, day)
		dateString := date.Format(time.DateOnly)
		minutes, err := loadMinutesForDate(dateString, goalID, userID)
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

func loadCalendarStats(goalID int, userID int, days int) ([]DailyStat, error) {
	if days <= 0 {
		days = 42
	}

	today := dateOnly(time.Now())
	start := today.AddDate(0, 0, -(days - 1))
	stats := make([]DailyStat, 0, days)

	for day := 0; day < days; day++ {
		date := start.AddDate(0, 0, day)
		dateString := date.Format(time.DateOnly)
		minutes, target, completed, err := loadProgressForDate(dateString, goalID, userID)
		if err != nil {
			return nil, err
		}

		stats = append(stats, DailyStat{
			Date:          dateString,
			Label:         weekdayLabel(date),
			Minutes:       minutes,
			TargetMinutes: target,
			IsCompleted:   completed,
		})
	}

	return stats, nil
}

func loadProgressForDate(date string, goalID int, userID int) (int, int, bool, error) {
	var minutes int
	var target int
	var completedCount int
	var totalCount int

	query := `
		SELECT
			COALESCE(SUM(total_minutes), 0),
			COALESCE(SUM(target_minutes), 0),
			COALESCE(SUM(is_completed), 0),
			COUNT(*)
		FROM daily_progress
		INNER JOIN goals ON goals.id = daily_progress.goal_id
		WHERE date = ? AND goals.user_id = ?
	`
	args := []any{date, userID}
	if goalID != 0 {
		query += ` AND goal_id = ?`
		args = append(args, goalID)
	}

	err := db.QueryRow(query, args...).Scan(&minutes, &target, &completedCount, &totalCount)
	if err != nil {
		return 0, 0, false, err
	}

	return minutes, target, totalCount > 0 && completedCount == totalCount, nil
}

func loadCompletionSummary(goalID int, userID int, startDate string, endDate string) (int, int, int, error) {
	var completed int
	var total int

	query := `
		SELECT COALESCE(SUM(is_completed), 0), COUNT(*)
		FROM daily_progress
		INNER JOIN goals ON goals.id = daily_progress.goal_id
		WHERE (date < ? OR is_completed = 1) AND goals.user_id = ?
	`
	args := []any{todayString(), userID}
	if goalID != 0 {
		query += ` AND goal_id = ?`
		args = append(args, goalID)
	}
	if startDate != "" {
		query += ` AND date >= ?`
		args = append(args, startDate)
	}
	if endDate != "" {
		query += ` AND date <= ?`
		args = append(args, endDate)
	}

	if err := db.QueryRow(query, args...).Scan(&completed, &total); err != nil {
		return 0, 0, 0, err
	}

	missed := max(total-completed, 0)
	return completed, missed, percent(completed, total), nil
}

func loadMinutesBetween(startDate string, endDate string, goalID int, userID int) (int, error) {
	var minutes int
	query := `
		SELECT COALESCE(SUM(total_minutes), 0)
		FROM daily_progress
		INNER JOIN goals ON goals.id = daily_progress.goal_id
		WHERE date >= ? AND date <= ? AND goals.user_id = ?
	`
	args := []any{startDate, endDate, userID}
	if goalID != 0 {
		query += ` AND goal_id = ?`
		args = append(args, goalID)
	}

	err := db.QueryRow(query, args...).Scan(&minutes)
	return minutes, err
}

func loadMonthlyTotal(goalID int, userID int) (int, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(time.DateOnly)
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Format(time.DateOnly)

	var minutes int
	query := `
		SELECT COALESCE(SUM(total_minutes), 0)
		FROM daily_progress
		INNER JOIN goals ON goals.id = daily_progress.goal_id
		WHERE date >= ? AND date < ? AND goals.user_id = ?
	`
	args := []any{monthStart, nextMonth, userID}
	if goalID != 0 {
		query += ` AND goal_id = ?`
		args = append(args, goalID)
	}

	err := db.QueryRow(query, args...).Scan(&minutes)
	return minutes, err
}

func loadGoalDistribution(totalMinutes int, goalID int, userID int) ([]GoalDistribution, error) {
	query := `
		SELECT goals.id, goals.title, COALESCE(SUM(daily_progress.total_minutes), 0)
		FROM goals
		LEFT JOIN daily_progress ON daily_progress.goal_id = goals.id
		WHERE goals.user_id = ?
	`
	args := []any{userID}
	if goalID != 0 {
		query += ` AND goals.id = ?`
		args = append(args, goalID)
	}
	query += `
		GROUP BY goals.id, goals.title
		ORDER BY COALESCE(SUM(daily_progress.total_minutes), 0) DESC
	`

	rows, err := db.Query(query, args...)
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

func parseDateOnlyOrToday(value string) time.Time {
	date, err := time.ParseInLocation(time.DateOnly, value, time.Local)
	if err != nil {
		return dateOnly(time.Now())
	}
	return date
}

func dateOnly(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
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

func signedPercentChange(current int, previous int) int {
	if previous <= 0 {
		if current > 0 {
			return 100
		}
		return 0
	}

	return int(math.Round(((float64(current) - float64(previous)) / float64(previous)) * 100))
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
