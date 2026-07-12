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

	wrongMethod := apiRequest(t, router, http.MethodPost, "/health", "", nil)
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d, want %d", wrongMethod.Code, http.StatusMethodNotAllowed)
	}
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
		{http.MethodPost, "/goals/1/sessions", `{"startedAt":"2026-07-12T10:00:00+03:00","endedAt":"2026-07-12T10:10:00+03:00","durationMinutes":10}`},
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
	createSessionForGoal(t, router, ownerCookie, goal.ID, 15)

	paths := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, fmt.Sprintf("/goals/%d", goal.ID), ""},
		{http.MethodPatch, fmt.Sprintf("/goals/%d", goal.ID), `{"title":"Hijacked","totalDays":10,"dailyTargetMinutes":10}`},
		{http.MethodDelete, fmt.Sprintf("/goals/%d", goal.ID), ""},
		{http.MethodPost, fmt.Sprintf("/goals/%d/sessions", goal.ID), sessionJSON(5)},
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
	secondResponse := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/sessions", goal.ID), sessionJSON(6), cookie)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("merged session status = %d, body = %s", secondResponse.Code, secondResponse.Body.String())
	}
	var merged Session
	decodeResponse(t, secondResponse, &merged)
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
		{http.MethodPost, fmt.Sprintf("/goals/%d/sessions", goal.ID), `{"startedAt":"","endedAt":"","durationMinutes":0}`, http.StatusBadRequest},
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

func apiRequest(t *testing.T, router http.Handler, method string, path string, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func registerAPIUser(t *testing.T, router http.Handler, email string) *http.Cookie {
	t.Helper()

	body := fmt.Sprintf(`{"email":%q,"name":"Test User","password":"Password123!"}`, email)
	response := apiRequest(t, router, http.MethodPost, "/auth/register", body, nil)
	if response.Code != http.StatusCreated {
		t.Fatalf("register %s status = %d, body = %s", email, response.Code, response.Body.String())
	}
	return authCookie(t, response)
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

	response := apiRequest(t, router, http.MethodPost, fmt.Sprintf("/goals/%d/sessions", goalID), sessionJSON(minutes), cookie)
	if response.Code != http.StatusCreated {
		t.Fatalf("create session status = %d, body = %s", response.Code, response.Body.String())
	}
	var session Session
	decodeResponse(t, response, &session)
	return session
}

func sessionJSON(minutes int) string {
	start := time.Now().In(time.Local).Truncate(time.Minute)
	end := start.Add(time.Duration(minutes) * time.Minute)
	return fmt.Sprintf(`{"startedAt":%q,"endedAt":%q,"durationMinutes":%d,"notes":"Practice","tags":["API"]}`,
		start.Format(time.RFC3339), end.Format(time.RFC3339), minutes)
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
