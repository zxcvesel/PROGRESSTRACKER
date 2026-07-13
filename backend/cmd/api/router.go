package main

import "net/http"

func newRouter() http.Handler {
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

	return securityMiddleware(mux)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}
