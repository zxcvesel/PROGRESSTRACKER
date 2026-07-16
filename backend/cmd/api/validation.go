package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxJSONBodyBytes   = 64 * 1024
	maxEmailLength     = 254
	maxNameLength      = 100
	maxPasswordLength  = 128
	maxGoalTitleLength = 120
	maxDescription     = 4000
	maxGoalDays        = 3650
	maxDailyMinutes    = 24 * 60
	maxNotesLength     = 4000
	maxTags            = 12
	maxTagLength       = 50
	maxCategoryLength  = 100
)

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			writeError(w, "request body is too large", http.StatusRequestEntityTooLarge)
			return false
		}
		writeError(w, "invalid JSON", http.StatusBadRequest)
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, "request body must contain a single JSON object", http.StatusBadRequest)
		return false
	}
	return true
}

func normalizeAndValidateEmail(value string) (string, bool) {
	email := normalizeEmail(value)
	if email == "" || len(email) > maxEmailLength {
		return "", false
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(parsed.Address, email) {
		return "", false
	}
	return email, true
}

func validPasswordLength(password string) bool {
	return len(password) >= 8 && len(password) <= maxPasswordLength
}

func validTextLength(value string, maximum int) bool {
	return utf8.ValidString(value) && utf8.RuneCountInString(value) <= maximum
}

func validateGoalInput(title *string, description *string, totalDays int, dailyMinutes int) string {
	*title = strings.TrimSpace(*title)
	*description = strings.TrimSpace(*description)
	if *title == "" || !validTextLength(*title, maxGoalTitleLength) {
		return "title must be between 1 and 120 characters"
	}
	if !validTextLength(*description, maxDescription) {
		return "description must not exceed 4000 characters"
	}
	if totalDays <= 0 || totalDays > maxGoalDays {
		return "totalDays must be between 1 and 3650"
	}
	if dailyMinutes <= 0 || dailyMinutes > maxDailyMinutes {
		return "dailyTargetMinutes must be between 1 and 1440"
	}
	return ""
}

func validateWeekdays(weekdays []int) bool {
	seen := make(map[int]bool)
	for _, weekday := range weekdays {
		if weekday < 1 || weekday > 7 || seen[weekday] {
			return false
		}
		seen[weekday] = true
	}
	return true
}

func validateSessionContent(notes string, tags []string) string {
	if !validTextLength(notes, maxNotesLength) {
		return "notes must not exceed 4000 characters"
	}
	if len(tags) > maxTags {
		return "no more than 12 tags are allowed"
	}
	for _, tag := range tags {
		if !validTextLength(tag, maxTagLength) {
			return "each tag must not exceed 50 characters"
		}
	}
	return ""
}

func validateEntryInput(entry *Entry) string {
	entry.Date = strings.TrimSpace(entry.Date)
	entry.Category = strings.TrimSpace(entry.Category)
	entry.Note = strings.TrimSpace(entry.Note)
	if _, err := time.Parse(time.DateOnly, entry.Date); err != nil {
		return "date must use YYYY-MM-DD"
	}
	if entry.Category == "" || !validTextLength(entry.Category, maxCategoryLength) {
		return "category must be between 1 and 100 characters"
	}
	if entry.Minutes <= 0 || entry.Minutes > maxDailyMinutes {
		return "minutes must be between 1 and 1440"
	}
	if !validTextLength(entry.Note, maxNotesLength) {
		return "note must not exceed 4000 characters"
	}
	return ""
}
