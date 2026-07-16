package main

import "net/http"

func entriesHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}
	rows, err := db.Query(`
		SELECT id, date, category, minutes, note
		FROM entries WHERE user_id = ? ORDER BY date DESC, id DESC
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
	user, ok := currentVerifiedUserFromRequest(w, r)
	if !ok {
		return
	}
	var entry Entry
	if !decodeJSON(w, r, &entry) {
		return
	}
	if message := validateEntryInput(&entry); message != "" {
		writeError(w, message, http.StatusBadRequest)
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
