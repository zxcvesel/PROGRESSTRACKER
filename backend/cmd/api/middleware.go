package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

func securityMiddleware(next http.Handler) http.Handler {
	allowedOrigins := loadAllowedOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		if secureCookiesEnabled() {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		if requiresOriginCheck(r) && !originAllowed(r, allowedOrigins) {
			writeError(w, "request origin is not allowed", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func requiresOriginCheck(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return false
	}
	_, err := r.Cookie(authCookieName)
	return err == nil
}

func originAllowed(r *http.Request, allowedOrigins map[string]bool) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	if allowedOrigins[origin] {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil || parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return false
	}
	return strings.EqualFold(parsedOrigin.Host, r.Host)
}

func loadAllowedOrigins() map[string]bool {
	configured := os.Getenv("PROGRESS_TRACKER_ALLOWED_ORIGINS")
	if configured == "" {
		configured = "http://127.0.0.1:5173,http://localhost:5173"
	}

	origins := make(map[string]bool)
	for _, origin := range strings.Split(configured, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			origins[strings.TrimSuffix(trimmed, "/")] = true
		}
	}
	return origins
}

func secureCookiesEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PROGRESS_TRACKER_SECURE_COOKIES")), "true")
}
