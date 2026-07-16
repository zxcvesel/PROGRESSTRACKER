package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" || len(requestID) > 100 {
			requestID, _ = randomToken(12)
		}
		w.Header().Set("X-Request-ID", requestID)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		started := time.Now()
		next.ServeHTTP(recorder, r)
		entry, _ := json.Marshal(map[string]any{
			"type":          "http_request",
			"requestId":     requestID,
			"method":        r.Method,
			"path":          r.URL.Path,
			"status":        recorder.status,
			"durationMs":    time.Since(started).Milliseconds(),
			"remoteAddress": r.RemoteAddr,
		})
		log.Print(string(entry))
	})
}
