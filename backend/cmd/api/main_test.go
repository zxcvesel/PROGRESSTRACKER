package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestDatabase(t *testing.T) {
	t.Helper()

	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}

	previousLocation := time.Local
	time.Local = location
	dbPath := t.TempDir() + "/progress.db"
	t.Cleanup(func() {
		time.Local = previousLocation
		if db != nil {
			db.Close()
			db = nil
		}
	})

	t.Setenv("PROGRESS_TRACKER_DB_PATH", dbPath)

	database, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	db = database
}

func TestCalculateGoalStreaksUsesCalendarDays(t *testing.T) {
	setupTestDatabase(t)

	today := time.Now().In(time.Local)
	yesterday := today.AddDate(0, 0, -1)

	goal := Goal{
		ID:                 1,
		Title:              "Streak test",
		TotalDays:          30,
		DailyTargetMinutes: 5,
		StartDate:          yesterday.Format(time.DateOnly),
		Status:             "active",
	}

	_, err := db.Exec(`
		INSERT INTO goals (id, title, description, total_days, daily_target_minutes, active_weekdays, start_date, created_at, status)
		VALUES (?, ?, '', ?, ?, '1,2,3,4,5,6,7', ?, ?, 'active')
	`, goal.ID, goal.Title, goal.TotalDays, goal.DailyTargetMinutes, goal.StartDate, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	insertSessionForDate(t, goal.ID, yesterday, 5)
	insertSessionForDate(t, goal.ID, today, 5)

	current, longest, err := calculateGoalStreaks(goal)
	if err != nil {
		t.Fatal(err)
	}

	if current != 2 {
		t.Fatalf("current streak = %d, want 2", current)
	}
	if longest != 2 {
		t.Fatalf("longest streak = %d, want 2", longest)
	}
}

func TestCurrentStreakSurvivesUntilTodayEnds(t *testing.T) {
	setupTestDatabase(t)

	today := time.Now().In(time.Local)
	yesterday := today.AddDate(0, 0, -1)
	twoDaysAgo := today.AddDate(0, 0, -2)

	goal := insertTestGoal(t, Goal{
		ID:                 1,
		Title:              "Today pending",
		TotalDays:          30,
		DailyTargetMinutes: 10,
		StartDate:          twoDaysAgo.Format(time.DateOnly),
		Status:             "active",
	})

	insertSessionForDate(t, goal.ID, twoDaysAgo, 10)
	insertSessionForDate(t, goal.ID, yesterday, 10)

	current, longest, err := calculateGoalStreaks(goal)
	if err != nil {
		t.Fatal(err)
	}

	if current != 2 {
		t.Fatalf("current streak before today's session = %d, want 2", current)
	}
	if longest != 2 {
		t.Fatalf("longest streak before today's session = %d, want 2", longest)
	}

	insertSessionForDate(t, goal.ID, today, 10)

	current, longest, err = calculateGoalStreaks(goal)
	if err != nil {
		t.Fatal(err)
	}

	if current != 3 {
		t.Fatalf("current streak after today's completed session = %d, want 3", current)
	}
	if longest != 3 {
		t.Fatalf("longest streak after today's completed session = %d, want 3", longest)
	}
}

func TestCurrentStreakResetsAfterMissedDay(t *testing.T) {
	setupTestDatabase(t)

	today := time.Now().In(time.Local)
	twoDaysAgo := today.AddDate(0, 0, -2)
	threeDaysAgo := today.AddDate(0, 0, -3)

	goal := insertTestGoal(t, Goal{
		ID:                 1,
		Title:              "Missed day",
		TotalDays:          30,
		DailyTargetMinutes: 10,
		StartDate:          threeDaysAgo.Format(time.DateOnly),
		Status:             "active",
	})

	insertSessionForDate(t, goal.ID, threeDaysAgo, 10)
	insertSessionForDate(t, goal.ID, twoDaysAgo, 10)

	current, longest, err := calculateGoalStreaks(goal)
	if err != nil {
		t.Fatal(err)
	}

	if current != 0 {
		t.Fatalf("current streak after missed yesterday = %d, want 0", current)
	}
	if longest != 2 {
		t.Fatalf("longest streak after missed yesterday = %d, want 2", longest)
	}
}

func TestCreateSessionMergesSameDaySession(t *testing.T) {
	setupTestDatabase(t)

	_, err := db.Exec(`
		INSERT INTO goals (id, title, description, total_days, daily_target_minutes, active_weekdays, start_date, created_at, status)
		VALUES (1, 'Merge test', '', 30, 10, '1,2,3,4,5,6,7', ?, ?, 'active')
	`, todayString(), time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	for _, body := range []string{
		`{"startedAt":"2026-07-01T10:00:00+03:00","endedAt":"2026-07-01T10:04:00+03:00","durationMinutes":4,"notes":"first","tags":["api"]}`,
		`{"startedAt":"2026-07-01T13:00:00+03:00","endedAt":"2026-07-01T13:06:00+03:00","durationMinutes":6,"notes":"second","tags":["stats"]}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/goals/1/sessions", strings.NewReader(body))
		request.SetPathValue("id", "1")
		response := httptest.NewRecorder()

		createSessionHandler(response, request)

		if response.Code != http.StatusCreated && response.Code != http.StatusOK {
			t.Fatalf("createSessionHandler status = %d, body = %s", response.Code, response.Body.String())
		}
	}

	sessions, err := loadSessions(1, 12)
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 1 {
		t.Fatalf("sessions count = %d, want 1", len(sessions))
	}
	if sessions[0].DurationMinutes != 10 {
		t.Fatalf("duration = %d, want 10", sessions[0].DurationMinutes)
	}
	if !strings.Contains(sessions[0].Notes, "first") || !strings.Contains(sessions[0].Notes, "second") {
		t.Fatalf("merged notes = %q, want both notes", sessions[0].Notes)
	}
}

func TestGoalSummaryCountsCompletedDays(t *testing.T) {
	setupTestDatabase(t)

	goal := Goal{
		ID:                 1,
		Title:              "Progress test",
		TotalDays:          30,
		DailyTargetMinutes: 10,
		StartDate:          todayString(),
		Status:             "active",
	}

	_, err := db.Exec(`
		INSERT INTO goals (id, title, description, total_days, daily_target_minutes, active_weekdays, start_date, created_at, status)
		VALUES (?, ?, '', ?, ?, '1,2,3,4,5,6,7', ?, ?, 'active')
	`, goal.ID, goal.Title, goal.TotalDays, goal.DailyTargetMinutes, goal.StartDate, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	insertSessionForDate(t, goal.ID, time.Now(), 5)

	summary, err := buildGoalSummary(goal)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CurrentDay != 0 {
		t.Fatalf("current day after partial work = %d, want 0", summary.CurrentDay)
	}
	if summary.CurrentStreak != 0 {
		t.Fatalf("streak after partial work = %d, want 0", summary.CurrentStreak)
	}

	insertSessionForDate(t, goal.ID, time.Now(), 5)

	summary, err = buildGoalSummary(goal)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CurrentDay != 1 {
		t.Fatalf("current day after completed target = %d, want 1", summary.CurrentDay)
	}
	if summary.CurrentStreak != 1 {
		t.Fatalf("streak after completed target = %d, want 1", summary.CurrentStreak)
	}
}

func insertSessionForDate(t *testing.T, goalID int, date time.Time, durationMinutes int) {
	t.Helper()

	startedAt := time.Date(date.Year(), date.Month(), date.Day(), 20, 0, 0, 0, time.Local)
	endedAt := startedAt.Add(time.Duration(durationMinutes) * time.Minute)

	_, err := db.Exec(`
		INSERT INTO sessions (goal_id, started_at, ended_at, duration_minutes, notes, tags, created_at)
		VALUES (?, ?, ?, ?, '', '', ?)
	`, goalID, startedAt.Format(time.RFC3339), endedAt.Format(time.RFC3339), durationMinutes, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	goal, err := loadGoal(goalID)
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshDailyProgressForGoal(goal); err != nil {
		t.Fatal(err)
	}
}

func insertTestGoal(t *testing.T, goal Goal) Goal {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO goals (id, title, description, total_days, daily_target_minutes, active_weekdays, start_date, created_at, status)
		VALUES (?, ?, '', ?, ?, '1,2,3,4,5,6,7', ?, ?, 'active')
	`, goal.ID, goal.Title, goal.TotalDays, goal.DailyTargetMinutes, goal.StartDate, time.Now().Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}

	return goal
}
