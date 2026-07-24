package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIRouterHealthAndMethodHandling(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	health := apiRequest(t, router, http.MethodGet, "/health", "", nil)
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, body = %s", health.Code, health.Body.String())
	}
	if contentType := health.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("health content type = %q, want application/json", contentType)
	}
	for header, expected := range map[string]string{
		"Cache-Control":          "no-store",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	} {
		if actual := health.Header().Get(header); actual != expected {
			t.Fatalf("%s = %q, want %q", header, actual, expected)
		}
	}

	wrongMethod := apiRequest(t, router, http.MethodPost, "/health", "", nil)
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d, want %d", wrongMethod.Code, http.StatusMethodNotAllowed)
	}
}

func TestAPIMutatingCookieRequestRequiresAllowedOrigin(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	cookie := registerAPIUser(t, router, "origin@example.com")

	request := httptest.NewRequest(http.MethodPost, "/goals", strings.NewReader(`{"title":"Blocked","totalDays":10,"dailyTargetMinutes":10}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://attacker.example")
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("malicious origin status = %d, body = %s", response.Code, response.Body.String())
	}
	assertAPIError(t, response, "request origin is not allowed")
}

func TestAPIProtectedRoutesRequireAuthentication(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/me", ""},
		{http.MethodGet, "/goals", ""},
		{http.MethodPost, "/goals", `{"title":"Private","totalDays":30,"dailyTargetMinutes":10}`},
		{http.MethodGet, "/goals/1", ""},
		{http.MethodPost, "/goals/1/timer/start", `{}`},
		{http.MethodGet, "/push/public-key", ""},
		{http.MethodPost, "/push/subscriptions", `{}`},
		{http.MethodDelete, "/push/subscriptions", `{}`},
		{http.MethodGet, "/stats", ""},
	}

	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			response := apiRequest(t, router, test.method, test.path, test.body, nil)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			assertAPIError(t, response, "authorization token is required")
		})
	}
}

func TestAPIAuthCookieLifecycle(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	register := apiRequest(t, router, http.MethodPost, "/auth/register", `{
		"email":" Learner@Example.COM ",
		"name":" Learner ",
		"password":"Password123!"
	}`, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", register.Code, register.Body.String())
	}

	cookie := authCookie(t, register)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("unexpected auth cookie attributes: %+v", cookie)
	}

	var auth AuthResponse
	decodeResponse(t, register, &auth)
	if auth.User.Email != "learner@example.com" || auth.User.Name != "Learner" {
		t.Fatalf("registered user = %+v", auth.User)
	}

	me := apiRequest(t, router, http.MethodGet, "/me", "", cookie)
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d, body = %s", me.Code, me.Body.String())
	}
	unverifiedGoals := apiRequest(t, router, http.MethodGet, "/goals", "", cookie)
	if unverifiedGoals.Code != http.StatusForbidden {
		t.Fatalf("unverified goals status = %d, want %d", unverifiedGoals.Code, http.StatusForbidden)
	}

	logout := apiRequest(t, router, http.MethodPost, "/auth/logout", "", cookie)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d, body = %s", logout.Code, logout.Body.String())
	}

	afterLogout := apiRequest(t, router, http.MethodGet, "/me", "", cookie)
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d, want %d", afterLogout.Code, http.StatusUnauthorized)
	}
}

func TestAPIAuthValidationAndDuplicateRegistration(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	tests := []struct {
		name string
		body string
	}{
		{"invalid email", `{"email":"invalid","password":"Password123!"}`},
		{"weak password", `{"email":"weak@example.com","password":"password"}`},
		{"invalid json", `{"email":`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := apiRequest(t, router, http.MethodPost, "/auth/register", test.body, nil)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}

	first := apiRequest(t, router, http.MethodPost, "/auth/register", `{
		"email":"duplicate@example.com","password":"Password123!"
	}`, nil)
	if first.Code != http.StatusCreated {
		t.Fatalf("first registration status = %d, body = %s", first.Code, first.Body.String())
	}

	duplicate := apiRequest(t, router, http.MethodPost, "/auth/register", `{
		"email":"DUPLICATE@example.com","password":"Password123!"
	}`, nil)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, body = %s", duplicate.Code, duplicate.Body.String())
	}
}

func TestAPIEmailVerificationAndPasswordReset(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()

	register := apiRequest(t, router, http.MethodPost, "/auth/register", `{
		"email":"lifecycle@example.com","name":"Lifecycle","password":"Password123!"
	}`, nil)
	if register.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", register.Code, register.Body.String())
	}
	var registered AuthResponse
	decodeResponse(t, register, &registered)
	if registered.User.EmailVerified || registered.DevelopmentToken == "" {
		t.Fatalf("registration response = %+v", registered)
	}
	oldCookie := authCookie(t, register)

	verify := apiRequest(t, router, http.MethodPost, "/auth/verify-email",
		fmt.Sprintf(`{"token":%q}`, registered.DevelopmentToken), nil)
	if verify.Code != http.StatusOK {
		t.Fatalf("verify status = %d, body = %s", verify.Code, verify.Body.String())
	}
	var verified AuthResponse
	decodeResponse(t, verify, &verified)
	if !verified.User.EmailVerified {
		t.Fatalf("verified user = %+v", verified.User)
	}
	reusedVerification := apiRequest(t, router, http.MethodPost, "/auth/verify-email",
		fmt.Sprintf(`{"token":%q}`, registered.DevelopmentToken), nil)
	if reusedVerification.Code != http.StatusBadRequest {
		t.Fatalf("reused verification status = %d", reusedVerification.Code)
	}

	forgot := apiRequest(t, router, http.MethodPost, "/auth/forgot-password", `{"email":"lifecycle@example.com"}`, nil)
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot password status = %d, body = %s", forgot.Code, forgot.Body.String())
	}
	var action ActionResponse
	decodeResponse(t, forgot, &action)
	if action.DevelopmentToken == "" {
		t.Fatal("development reset token was not returned")
	}

	reset := apiRequest(t, router, http.MethodPost, "/auth/reset-password",
		fmt.Sprintf(`{"token":%q,"newPassword":"NewPassword123!"}`, action.DevelopmentToken), nil)
	if reset.Code != http.StatusNoContent {
		t.Fatalf("reset password status = %d, body = %s", reset.Code, reset.Body.String())
	}
	reusedReset := apiRequest(t, router, http.MethodPost, "/auth/reset-password",
		fmt.Sprintf(`{"token":%q,"newPassword":"AnotherPassword123!"}`, action.DevelopmentToken), nil)
	if reusedReset.Code != http.StatusBadRequest {
		t.Fatalf("reused reset status = %d", reusedReset.Code)
	}

	me := apiRequest(t, router, http.MethodGet, "/me", "", oldCookie)
	if me.Code != http.StatusUnauthorized {
		t.Fatalf("old session after reset status = %d", me.Code)
	}
	oldLogin := apiRequest(t, router, http.MethodPost, "/auth/login",
		`{"email":"lifecycle@example.com","password":"Password123!"}`, nil)
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old password login status = %d", oldLogin.Code)
	}
	newLogin := apiRequest(t, router, http.MethodPost, "/auth/login",
		`{"email":"lifecycle@example.com","password":"NewPassword123!"}`, nil)
	if newLogin.Code != http.StatusOK {
		t.Fatalf("new password login status = %d, body = %s", newLogin.Code, newLogin.Body.String())
	}
}

func TestAPILogoutAllAndDeleteAccount(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	cookie := registerAPIUser(t, router, "delete-account@example.com")
	goal := createAPIGoal(t, router, cookie, "Private goal", 5)
	createSessionForGoal(t, router, cookie, goal.ID, 1)

	wrongDelete := apiRequest(t, router, http.MethodDelete, "/me", `{"password":"WrongPassword123!"}`, cookie)
	if wrongDelete.Code != http.StatusBadRequest {
		t.Fatalf("wrong password delete status = %d", wrongDelete.Code)
	}

	logoutAll := apiRequest(t, router, http.MethodDelete, "/me/sessions", "", cookie)
	if logoutAll.Code != http.StatusNoContent {
		t.Fatalf("logout all status = %d, body = %s", logoutAll.Code, logoutAll.Body.String())
	}
	if response := apiRequest(t, router, http.MethodGet, "/me", "", cookie); response.Code != http.StatusUnauthorized {
		t.Fatalf("session survived logout all: %d", response.Code)
	}

	login := apiRequest(t, router, http.MethodPost, "/auth/login",
		`{"email":"delete-account@example.com","password":"Password123!"}`, nil)
	if login.Code != http.StatusOK {
		t.Fatalf("login before delete status = %d, body = %s", login.Code, login.Body.String())
	}
	deleteCookie := authCookie(t, login)
	remove := apiRequest(t, router, http.MethodDelete, "/me", `{"password":"Password123!"}`, deleteCookie)
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete account status = %d, body = %s", remove.Code, remove.Body.String())
	}

	var users int
	var goals int
	var sessions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = 'delete-account@example.com'`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM goals WHERE id = ?`, goal.ID).Scan(&goals); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE goal_id = ?`, goal.ID).Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if users != 0 || goals != 0 || sessions != 0 {
		t.Fatalf("account data remained: users=%d goals=%d sessions=%d", users, goals, sessions)
	}
}

func TestAPIAccountExportIsScopedAndSupportsCSV(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	first := registerAPIUser(t, router, "export-one@example.com")
	second := registerAPIUser(t, router, "export-two@example.com")
	firstGoal := createAPIGoal(t, router, first, "Exported goal", 30)
	createSessionForGoal(t, router, first, firstGoal.ID, 12)
	secondGoal := createAPIGoal(t, router, second, "Private goal", 30)
	createSessionForGoal(t, router, second, secondGoal.ID, 15)

	jsonExport := apiRequest(t, router, http.MethodGet, "/me/export", "", first)
	if jsonExport.Code != http.StatusOK {
		t.Fatalf("JSON export status = %d, body = %s", jsonExport.Code, jsonExport.Body.String())
	}
	if !strings.Contains(jsonExport.Body.String(), "Exported goal") || strings.Contains(jsonExport.Body.String(), "Private goal") {
		t.Fatalf("JSON export is not account-scoped: %s", jsonExport.Body.String())
	}

	csvExport := apiRequest(t, router, http.MethodGet, "/me/export?format=csv", "", first)
	if csvExport.Code != http.StatusOK || !strings.Contains(csvExport.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("CSV export status = %d, content-type = %s", csvExport.Code, csvExport.Header().Get("Content-Type"))
	}
	if !strings.Contains(csvExport.Body.String(), "Exported goal") || strings.Contains(csvExport.Body.String(), "Private goal") {
		t.Fatalf("CSV export is not account-scoped: %s", csvExport.Body.String())
	}
}

func TestAPIGoalLifecycleDeletesRelatedData(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	cookie := registerAPIUser(t, router, "goals@example.com")

	create := apiRequest(t, router, http.MethodPost, "/goals", `{
		"title":"Learn Go",
		"description":"Build APIs",
		"totalDays":30,
		"dailyTargetMinutes":20
	}`, cookie)
	if create.Code != http.StatusCreated {
		t.Fatalf("create goal status = %d, body = %s", create.Code, create.Body.String())
	}
	var goal GoalSummary
	decodeResponse(t, create, &goal)
	if goal.StartDate != todayString() || len(goal.ActiveWeekdays) != 7 || goal.Status != "active" {
		t.Fatalf("created goal defaults = %+v", goal.Goal)
	}

	updatePath := fmt.Sprintf("/goals/%d", goal.ID)
	update := apiRequest(t, router, http.MethodPatch, updatePath, `{
		"title":"Master Go",
		"description":"Production APIs",
		"totalDays":60,
		"dailyTargetMinutes":30,
		"status":"active"
	}`, cookie)
	if update.Code != http.StatusOK {
		t.Fatalf("update goal status = %d, body = %s", update.Code, update.Body.String())
	}
	var updated GoalSummary
	decodeResponse(t, update, &updated)
	if updated.Title != "Master Go" || updated.TotalDays != 60 || updated.DailyTargetMinutes != 30 {
		t.Fatalf("updated goal = %+v", updated.Goal)
	}

	createSessionForGoal(t, router, cookie, goal.ID, 30)

	remove := apiRequest(t, router, http.MethodDelete, updatePath, "", cookie)
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete goal status = %d, body = %s", remove.Code, remove.Body.String())
	}

	missing := apiRequest(t, router, http.MethodGet, updatePath, "", cookie)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("deleted goal status = %d, want %d", missing.Code, http.StatusNotFound)
	}

	for table, query := range map[string]string{
		"goals":          `SELECT COUNT(*) FROM goals WHERE id = ?`,
		"sessions":       `SELECT COUNT(*) FROM sessions WHERE goal_id = ?`,
		"daily_progress": `SELECT COUNT(*) FROM daily_progress WHERE goal_id = ?`,
	} {
		var count int
		if err := db.QueryRow(query, goal.ID).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows after goal deletion = %d, want 0", table, count)
		}
	}
}

func TestAPIAccountsCannotAccessEachOthersGoalsOrStats(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	ownerCookie := registerAPIUser(t, router, "owner@example.com")
	otherCookie := registerAPIUser(t, router, "other@example.com")

	goal := createAPIGoal(t, router, ownerCookie, "Owner goal", 15)
	session := createSessionForGoal(t, router, ownerCookie, goal.ID, 15)
	otherGoal := createAPIGoal(t, router, otherCookie, "Other goal", 15)

	paths := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, fmt.Sprintf("/goals/%d", goal.ID), ""},
		{http.MethodPatch, fmt.Sprintf("/goals/%d", goal.ID), `{"title":"Hijacked","totalDays":10,"dailyTargetMinutes":10}`},
		{http.MethodDelete, fmt.Sprintf("/goals/%d", goal.ID), ""},
		{http.MethodPost, fmt.Sprintf("/goals/%d/timer/start", goal.ID), `{}`},
		{http.MethodPost, fmt.Sprintf("/goals/%d/timer/pause", goal.ID), ""},
		{http.MethodPost, fmt.Sprintf("/goals/%d/timer/resume", goal.ID), ""},
		{http.MethodPost, fmt.Sprintf("/goals/%d/timer/finish", goal.ID), `{}`},
		{http.MethodPatch, fmt.Sprintf("/goals/%d/sessions/%d", goal.ID, session.ID), `{"notes":"Hijacked"}`},
		{http.MethodDelete, fmt.Sprintf("/goals/%d/sessions/%d", goal.ID, session.ID), ""},
		{http.MethodPatch, fmt.Sprintf("/goals/%d/sessions/%d", otherGoal.ID, session.ID), `{"notes":"Hijacked"}`},
		{http.MethodDelete, fmt.Sprintf("/goals/%d/sessions/%d", otherGoal.ID, session.ID), ""},
		{http.MethodGet, fmt.Sprintf("/stats?goalId=%d", goal.ID), ""},
	}

	for _, test := range paths {
		response := apiRequest(t, router, test.method, test.path, test.body, otherCookie)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, body = %s", test.method, test.path, response.Code, response.Body.String())
		}
	}

	ownerDetail := apiRequest(t, router, http.MethodGet, fmt.Sprintf("/goals/%d", goal.ID), "", ownerCookie)
	if ownerDetail.Code != http.StatusOK {
		t.Fatalf("owner goal disappeared: status = %d, body = %s", ownerDetail.Code, ownerDetail.Body.String())
	}
}

func TestAPISessionLifecycleRecalculatesProgressAndStats(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	cookie := registerAPIUser(t, router, "sessions@example.com")
	goal := createAPIGoal(t, router, cookie, "Daily API", 10)

	first := createSessionForGoal(t, router, cookie, goal.ID, 4)
	secondResponse, merged := finishServerTimerForGoal(t, router, cookie, goal.ID, 6)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("merged session status = %d, body = %s", secondResponse.Code, secondResponse.Body.String())
	}
	if merged.ID != first.ID || merged.DurationMinutes != 10 {
		t.Fatalf("merged session = %+v", merged)
	}

	statsResponse := apiRequest(t, router, http.MethodGet, fmt.Sprintf("/stats?goalId=%d", goal.ID), "", cookie)
	if statsResponse.Code != http.StatusOK {
		t.Fatalf("stats status = %d, body = %s", statsResponse.Code, statsResponse.Body.String())
	}
	var stats Stats
	decodeResponse(t, statsResponse, &stats)
	if stats.TotalSessions != 1 || stats.TotalPracticeMinutes != 10 || stats.TodayMinutes != 10 || stats.CompletedDays != 1 || stats.CurrentStreak != 1 {
		t.Fatalf("stats after completed target = %+v", stats)
	}

	update := apiRequest(t, router, http.MethodPatch,
		fmt.Sprintf("/goals/%d/sessions/%d", goal.ID, merged.ID),
		`{"notes":"Updated note","tags":["Go","API"]}`, cookie)
	if update.Code != http.StatusOK {
		t.Fatalf("update session status = %d, body = %s", update.Code, update.Body.String())
	}
	var updated Session
	decodeResponse(t, update, &updated)
	if updated.Notes != "Updated note" || len(updated.Tags) != 2 {
		t.Fatalf("updated session = %+v", updated)
	}

	remove := apiRequest(t, router, http.MethodDelete,
		fmt.Sprintf("/goals/%d/sessions/%d", goal.ID, merged.ID), "", cookie)
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete session status = %d, body = %s", remove.Code, remove.Body.String())
	}

	statsAfterDelete := apiRequest(t, router, http.MethodGet, fmt.Sprintf("/stats?goalId=%d", goal.ID), "", cookie)
	var emptyStats Stats
	decodeResponse(t, statsAfterDelete, &emptyStats)
	if emptyStats.TotalSessions != 0 || emptyStats.TotalPracticeMinutes != 0 || emptyStats.TodayMinutes != 0 || emptyStats.CompletedDays != 0 || emptyStats.CurrentStreak != 0 {
		t.Fatalf("stats after session deletion = %+v", emptyStats)
	}
}

func TestAPIServerTimerLifecycle(t *testing.T) {
	setupTestDatabase(t)
	t.Setenv("PROGRESS_TRACKER_DEV_TIMER_SPEED", "true")
	router := newRouter()
	cookie := registerAPIUser(t, router, "timer@example.com")
	goal := createAPIGoal(t, router, cookie, "Server timer", 10)
	otherGoal := createAPIGoal(t, router, cookie, "Other goal", 10)

	start := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/start", goal.ID), `{"speedMultiplier":5}`, cookie)
	if start.Code != http.StatusCreated {
		t.Fatalf("start timer status = %d, body = %s", start.Code, start.Body.String())
	}
	var started TimerState
	decodeResponse(t, start, &started)
	if started.State != "running" || started.GoalID != goal.ID || started.TargetSeconds != 600 || started.SpeedMultiplier != 5 {
		t.Fatalf("started timer = %+v", started)
	}

	conflict := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/start", otherGoal.ID), `{}`, cookie)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("second timer status = %d, body = %s", conflict.Code, conflict.Body.String())
	}

	pause := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/pause", goal.ID), "", cookie)
	if pause.Code != http.StatusOK {
		t.Fatalf("pause timer status = %d, body = %s", pause.Code, pause.Body.String())
	}
	var paused TimerState
	decodeResponse(t, pause, &paused)
	if paused.State != "paused" {
		t.Fatalf("paused timer = %+v", paused)
	}

	active := apiRequest(t, router, http.MethodGet, "/timer", "", cookie)
	var status TimerStatusResponse
	decodeResponse(t, active, &status)
	if !status.Active || status.Timer == nil || status.Timer.State != "paused" {
		t.Fatalf("active timer response = %+v", status)
	}

	resume := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/resume", goal.ID), "", cookie)
	if resume.Code != http.StatusOK {
		t.Fatalf("resume timer status = %d, body = %s", resume.Code, resume.Body.String())
	}

	forged := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/finish", goal.ID),
		`{"notes":"Practice","tags":["Timer"],"durationMinutes":600}`, cookie)
	if forged.Code != http.StatusBadRequest {
		t.Fatalf("forged duration status = %d, body = %s", forged.Code, forged.Body.String())
	}

	finish := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/finish", goal.ID),
		`{"notes":"Server measured practice","tags":["Timer"]}`, cookie)
	if finish.Code != http.StatusCreated {
		t.Fatalf("finish timer status = %d, body = %s", finish.Code, finish.Body.String())
	}
	var session Session
	decodeResponse(t, finish, &session)
	if session.DurationMinutes != 1 || session.Notes != "Server measured practice" {
		t.Fatalf("finished timer session = %+v", session)
	}

	inactive := apiRequest(t, router, http.MethodGet, "/timer", "", cookie)
	var inactiveStatus TimerStatusResponse
	decodeResponse(t, inactive, &inactiveStatus)
	if inactiveStatus.Active || inactiveStatus.Timer != nil {
		t.Fatalf("timer remained active after finish: %+v", inactiveStatus)
	}
}

func TestAPIServerTimerSpeedIsDevelopmentOnly(t *testing.T) {
	setupTestDatabase(t)
	t.Setenv("PROGRESS_TRACKER_HOST", "0.0.0.0")
	router := newRouter()
	cookie := registerAPIUser(t, router, "timer-production@example.com")
	goal := createAPIGoal(t, router, cookie, "Production timer", 10)

	response := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/start", goal.ID), `{"speedMultiplier":5}`, cookie)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("production speed status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestAPIRejectsInvalidResourceInput(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	cookie := registerAPIUser(t, router, "validation@example.com")
	goal := createAPIGoal(t, router, cookie, "Validation", 10)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{http.MethodPost, "/goals", `{"title":"","totalDays":0,"dailyTargetMinutes":0}`, http.StatusBadRequest},
		{http.MethodGet, "/goals/not-a-number", "", http.StatusBadRequest},
		{http.MethodPatch, fmt.Sprintf("/goals/%d", goal.ID), `{"title":"Goal","totalDays":10,"dailyTargetMinutes":10,"status":"unknown"}`, http.StatusBadRequest},
		{http.MethodPost, fmt.Sprintf("/goals/%d/timer/start", goal.ID), `{"speedMultiplier":3}`, http.StatusBadRequest},
		{http.MethodPatch, fmt.Sprintf("/goals/%d/sessions/not-a-number", goal.ID), `{}`, http.StatusBadRequest},
		{http.MethodGet, "/stats?goalId=invalid", "", http.StatusBadRequest},
		{http.MethodGet, "/stats?goalId=-1", "", http.StatusBadRequest},
	}

	for _, test := range tests {
		response := apiRequest(t, router, test.method, test.path, test.body, cookie)
		if response.Code != test.status {
			t.Fatalf("%s %s status = %d, want %d, body = %s", test.method, test.path, response.Code, test.status, response.Body.String())
		}
	}
}

func TestAPIStrictJSONAndBodyLimit(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	cookie := registerAPIUser(t, router, "json@example.com")

	tests := []struct {
		name   string
		body   string
		status int
	}{
		{"unknown field", `{"title":"Goal","totalDays":10,"dailyTargetMinutes":10,"admin":true}`, http.StatusBadRequest},
		{"multiple objects", `{"title":"Goal","totalDays":10,"dailyTargetMinutes":10}{}`, http.StatusBadRequest},
		{"oversized body", `{"title":"Goal","description":"` + strings.Repeat("a", maxJSONBodyBytes) + `","totalDays":10,"dailyTargetMinutes":10}`, http.StatusRequestEntityTooLarge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := apiRequest(t, router, http.MethodPost, "/goals", test.body, cookie)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, test.status, response.Body.String())
			}
		})
	}
}

func TestRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	if !limiter.Allow("login:client") || !limiter.Allow("login:client") {
		t.Fatal("rate limiter rejected a request before the limit")
	}
	if limiter.Allow("login:client") {
		t.Fatal("rate limiter allowed a request over the limit")
	}
	if !limiter.Allow("login:other-client") {
		t.Fatal("rate limiter mixed independent clients")
	}
}

func TestRateLimitKeyTrustsForwardedAddressOnlyWhenConfigured(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	request.RemoteAddr = "192.0.2.10:54321"
	request.Header.Set("X-Forwarded-For", "198.51.100.8, 192.0.2.20")
	request.Header.Set("X-Real-IP", "203.0.113.25")

	if key := rateLimitKey(request, "login"); key != "login:192.0.2.10" {
		t.Fatalf("untrusted proxy key = %q", key)
	}
	t.Setenv("PROGRESS_TRACKER_TRUST_PROXY", "true")
	if key := rateLimitKey(request, "login"); key != "login:203.0.113.25" {
		t.Fatalf("trusted proxy key = %q", key)
	}
}

func TestLoginFailuresArePersistentlyLimited(t *testing.T) {
	setupTestDatabase(t)
	router := newRouter()
	registerAPIUser(t, router, "limited@example.com")

	now := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	loginAttempts.accountPolicy.limit = 3
	loginAttempts.clientPolicy.limit = 10
	loginAttempts.now = func() time.Time { return now }

	for attempt := 1; attempt <= 2; attempt++ {
		response := apiRequest(t, router, http.MethodPost, "/auth/login",
			`{"email":"limited@example.com","password":"WrongPassword123!"}`, nil)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("failed login %d status = %d, body = %s", attempt, response.Code, response.Body.String())
		}
	}

	blocked := apiRequest(t, router, http.MethodPost, "/auth/login",
		`{"email":"limited@example.com","password":"WrongPassword123!"}`, nil)
	if blocked.Code != http.StatusTooManyRequests || blocked.Header().Get("Retry-After") == "" {
		t.Fatalf("blocking response status = %d, retry-after = %q, body = %s",
			blocked.Code, blocked.Header().Get("Retry-After"), blocked.Body.String())
	}

	loginAttempts = newLoginAttemptLimiter()
	loginAttempts.now = func() time.Time { return now }
	stillBlocked := apiRequest(t, router, http.MethodPost, "/auth/login",
		`{"email":"limited@example.com","password":"Password123!"}`, nil)
	if stillBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("persistent block status = %d, body = %s", stillBlocked.Code, stillBlocked.Body.String())
	}

	now = now.Add(loginBlockDuration + time.Second)
	allowed := apiRequest(t, router, http.MethodPost, "/auth/login",
		`{"email":"limited@example.com","password":"Password123!"}`, nil)
	if allowed.Code != http.StatusOK {
		t.Fatalf("login after block status = %d, body = %s", allowed.Code, allowed.Body.String())
	}

	var accountRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM login_attempts WHERE key_hash = ?`,
		loginAttemptKey("account", "limited@example.com")).Scan(&accountRows); err != nil {
		t.Fatal(err)
	}
	if accountRows != 0 {
		t.Fatalf("successful login retained %d account blocks", accountRows)
	}
}

func TestDatabaseMigrationsAndConstraints(t *testing.T) {
	setupTestDatabase(t)

	var migrations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
		t.Fatal(err)
	}
	if migrations != len(databaseMigrations) {
		t.Fatalf("migration count = %d, want %d", migrations, len(databaseMigrations))
	}
	if err := runDatabaseMigrations(db); err != nil {
		t.Fatalf("repeat migrations: %v", err)
	}
	var migrationsAfterRepeat int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationsAfterRepeat); err != nil {
		t.Fatal(err)
	}
	if migrationsAfterRepeat != migrations {
		t.Fatalf("migration count after repeat = %d, want %d", migrationsAfterRepeat, migrations)
	}

	var foreignKeys int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	_, err := db.Exec(`
		INSERT INTO goals (title, description, total_days, daily_target_minutes, active_weekdays, start_date, created_at, status, user_id)
		VALUES ('Invalid', '', 0, 10, '1,2,3,4,5,6,7', ?, ?, 'active', 1)
	`, todayString(), time.Now().Format(time.RFC3339))
	if err == nil {
		t.Fatal("database accepted invalid goal duration")
	}
}

func apiRequest(t *testing.T, router http.Handler, method string, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
		if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
			request.Header.Set("Origin", "http://127.0.0.1:5173")
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func registerAPIUser(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()

	body := fmt.Sprintf(`{"email":%q,"name":"Test User","password":"Password123!","timezone":"Europe/Moscow"}`, email)
	response := apiRequest(t, router, http.MethodPost, "/auth/register", body, nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("register %s status = %d, body = %s", email, response.Code, response.Body.String())
	}
	cookie := authCookie(t, response)
	var registered AuthResponse
	decodeResponse(t, response, &registered)
	if registered.DevelopmentToken == "" {
		if _, err := db.Exec(`UPDATE users SET email_verified = 1 WHERE email = ?`, email); err != nil {
			t.Fatal(err)
		}
		return cookie
	}
	verification := apiRequest(t, router, http.MethodPost, "/auth/verify-email",
		fmt.Sprintf(`{"token":%q}`, registered.DevelopmentToken), nil)
	if verification.Code != http.StatusOK {
		t.Fatalf("verify %s status = %d, body = %s", email, verification.Code, verification.Body.String())
	}
	return cookie
}

func authCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == authCookieName {
			return cookie
		}
	}
	t.Fatal("auth cookie was not set")
	return nil
}

func createAPIGoal(t *testing.T, router http.Handler, cookie *http.Cookie, title string, targetMinutes int) GoalSummary {
	t.Helper()

	body := fmt.Sprintf(`{"title":%q,"totalDays":30,"dailyTargetMinutes":%d}`, title, targetMinutes)
	response := apiRequest(t, router, http.MethodPost, "/goals", body, cookie)
	if response.Code != http.StatusCreated {
		t.Fatalf("create goal status = %d, body = %s", response.Code, response.Body.String())
	}
	var goal GoalSummary
	decodeResponse(t, response, &goal)
	return goal
}

func createSessionForGoal(t *testing.T, router http.Handler, cookie *http.Cookie, goalID int, minutes int) Session {
	t.Helper()

	response, session := finishServerTimerForGoal(t, router, cookie, goalID, minutes)
	if response.Code != http.StatusCreated {
		t.Fatalf("create session status = %d, body = %s", response.Code, response.Body.String())
	}
	return session
}

func finishServerTimerForGoal(t *testing.T, router http.Handler, cookie *http.Cookie, goalID int, minutes int) (*httptest.ResponseRecorder, Session) {
	t.Helper()

	start := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/start", goalID), `{}`, cookie)
	if start.Code != http.StatusCreated {
		t.Fatalf("start timer status = %d, body = %s", start.Code, start.Body.String())
	}
	if _, err := db.Exec(`
		UPDATE active_timers
		SET state = 'paused', accumulated_seconds = ?
		WHERE goal_id = ?
	`, float64(minutes*60), goalID); err != nil {
		t.Fatal(err)
	}

	response := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/timer/finish", goalID),
		`{"notes":"Practice","tags":["API"]}`, cookie)
	if response.Code != http.StatusCreated && response.Code != http.StatusOK {
		t.Fatalf("finish timer status = %d, body = %s", response.Code, response.Body.String())
	}
	var session Session
	decodeResponse(t, response, &session)
	return response, session
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v, body = %s", err, response.Body.String())
	}
}

func assertAPIError(t *testing.T, response *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var apiError ErrorResponse
	decodeResponse(t, response, &apiError)
	if apiError.Error != expected {
		t.Fatalf("error = %q, want %q", apiError.Error, expected)
	}
}
