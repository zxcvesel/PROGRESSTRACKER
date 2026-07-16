package main

import (
	"net/http"
	"strings"
	"time"

	_ "time/tzdata"
)

func normalizeTimezone(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "UTC", true
	}
	if len(value) > 100 {
		return "", false
	}
	if _, err := time.LoadLocation(value); err != nil {
		return "", false
	}
	return value, true
}

func userLocation(userID int) *time.Location {
	var name string
	if err := db.QueryRow(`SELECT timezone FROM users WHERE id = ?`, userID).Scan(&name); err != nil {
		return time.UTC
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return location
}

func userNow(userID int) time.Time {
	return time.Now().In(userLocation(userID))
}

func todayStringForUser(userID int) string {
	return userNow(userID).Format(time.DateOnly)
}

func dateOnlyForUser(userID int, value time.Time) time.Time {
	location := userLocation(userID)
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

func parseDateForUser(userID int, value string) time.Time {
	location := userLocation(userID)
	date, err := time.ParseInLocation(time.DateOnly, value, location)
	if err != nil {
		return dateOnlyForUser(userID, time.Now())
	}
	return date
}

func updateTimezoneHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}
	var request UpdateTimezoneRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	timezone, valid := normalizeTimezone(request.Timezone)
	if !valid {
		writeError(w, "invalid timezone", http.StatusBadRequest)
		return
	}
	if _, err := db.Exec(`UPDATE users SET timezone = ? WHERE id = ?`, timezone, user.ID); err != nil {
		writeError(w, "failed to update timezone", http.StatusInternalServerError)
		return
	}
	updated, err := loadUserByID(user.ID)
	if err != nil {
		writeError(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	writeJSON(w, updated, http.StatusOK)
}
