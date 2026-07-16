package main

import "net/http"

func newRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readinessHandler)
	mux.HandleFunc("POST /auth/register", registerHandler)
	mux.HandleFunc("POST /auth/login", loginHandler)
	mux.HandleFunc("POST /auth/logout", logoutHandler)
	mux.HandleFunc("POST /auth/verify-email", verifyEmailHandler)
	mux.HandleFunc("POST /auth/resend-verification", resendVerificationHandler)
	mux.HandleFunc("POST /auth/forgot-password", forgotPasswordHandler)
	mux.HandleFunc("POST /auth/reset-password", resetPasswordHandler)
	mux.HandleFunc("GET /me", meHandler)
	mux.HandleFunc("PATCH /me", updateMeHandler)
	mux.HandleFunc("PATCH /me/timezone", updateTimezoneHandler)
	mux.HandleFunc("PATCH /me/password", changePasswordHandler)
	mux.HandleFunc("GET /me/export", exportAccountHandler)
	mux.HandleFunc("DELETE /me/sessions", logoutAllHandler)
	mux.HandleFunc("DELETE /me", deleteAccountHandler)
	mux.HandleFunc("GET /entries", entriesHandler)
	mux.HandleFunc("POST /entries", createEntryHandler)
	mux.HandleFunc("GET /goals", goalsHandler)
	mux.HandleFunc("POST /goals", createGoalHandler)
	mux.HandleFunc("GET /goals/{id}", goalDetailHandler)
	mux.HandleFunc("PATCH /goals/{id}", updateGoalHandler)
	mux.HandleFunc("DELETE /goals/{id}", deleteGoalHandler)
	mux.HandleFunc("PATCH /goals/{id}/sessions/{sessionId}", updateSessionHandler)
	mux.HandleFunc("DELETE /goals/{id}/sessions/{sessionId}", deleteSessionHandler)
	mux.HandleFunc("GET /timer", activeTimerHandler)
	mux.HandleFunc("POST /goals/{id}/timer/start", startTimerHandler)
	mux.HandleFunc("POST /goals/{id}/timer/pause", pauseTimerHandler)
	mux.HandleFunc("POST /goals/{id}/timer/resume", resumeTimerHandler)
	mux.HandleFunc("POST /goals/{id}/timer/finish", finishTimerHandler)
	mux.HandleFunc("GET /stats", statsHandler)

	return requestLoggingMiddleware(securityMiddleware(mux))
}

func readinessHandler(w http.ResponseWriter, _ *http.Request) {
	if db == nil || db.Ping() != nil {
		writeError(w, "database is not ready", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, map[string]string{"status": "ready"}, http.StatusOK)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}
